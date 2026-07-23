package mitsubishi

import (
	nats_go "github.com/nats-io/nats.go"
	nats_client "github.com/program-dg/dvc-gateway/internal/nats"

	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/program-dg/dvc-gateway/internal/nats"

	"gorm.io/gorm/clause"

	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/database/postgres/models"
)

type PollerStats struct {
	ReadOps        uint64
	ReadMicros     uint64
	WriteOps       uint64
	WriteMicros    uint64
	ErrorCount     int32
	PubCount       uint64
	RedisPubMicros uint64
	NatsPubMicros  uint64
}

// GetPollerStats safely returns the current snapshot of poller metrics
func GetPollerStats() PollerStats {
	return PollerStats{
		ReadOps:        atomic.LoadUint64(&plcStats.ReadOps),
		ReadMicros:     atomic.LoadUint64(&plcStats.ReadMicros),
		WriteOps:       atomic.LoadUint64(&plcStats.WriteOps),
		WriteMicros:    atomic.LoadUint64(&plcStats.WriteMicros),
		ErrorCount:     atomic.LoadInt32(&plcStats.ErrorCount),
		PubCount:       atomic.LoadUint64(&plcStats.PubCount),
		NatsPubMicros:  atomic.LoadUint64(&plcStats.NatsPubMicros),
	}
}

var (
	plcStats      PollerStats
	prevReadOps   uint64
	prevWriteOps  uint64
	fallbackPorts = []int{1033, 1037, 1040}
)

type PLCHealth struct {
	Status    string
	LatencyMS float64
	LastSeen  time.Time
}

var PlcHealthMap sync.Map

// Poll connects to the PLC and starts continuous data extraction
func Poll(plc models.MitsubishiPlc) {
	// Proactive port auto-scan on startup
	if foundPort, err := findMCPort(plc.IpAddress, plc.Port); err == nil {
		if foundPort != plc.Port {
			log.Printf("[Poller] auto-scan: %s responds on port %d (was %d), updating DB", plc.IpAddress, foundPort, plc.Port)
			plc.Port = foundPort
			postgres.DB.Model(&plc).Update("port", foundPort)
		}
	} else {
		log.Printf("[Poller] no MC responder on %s", plc.IpAddress)
	}

	// ✅ FIX: use MitsubishiTagList instead of PlcBenchTag + NO LIMIT
	var allTags []models.MitsubishiTagList

	err := postgres.DB.
		Preload("Plc").
		Where("plc_id = ?", plc.ID).
		Order("tag_address").
		Find(&allTags).Error

	if err != nil {
		log.Printf("[Poller] tag load error: %v", err)
		return
	}

	if len(allTags) == 0 {
		log.Printf("[Poller] No tags found for PLC %s, will start health check only", plc.IpAddress)
	}

	// Convert tags
	type tagInfo struct {
		device  string
		offset  int
		dbID    string // UUID
		tagName string
	}

	var tags []tagInfo

	for _, t := range allTags {
		i := 0
		addr := t.TagAddress

		for i < len(addr) && (addr[i] < '0' || addr[i] > '9') {
			i++
		}

		if i >= len(addr) {
			continue
		}

		off, _ := strconv.Atoi(addr[i:])

		tags = append(tags, tagInfo{
			device:  addr[:i],
			offset:  off,
			dbID:    t.ID,
			tagName: t.TagName,
		})
	}

	if len(tags) == 0 {
		log.Println("[Poller] No valid tags after parsing, adding dummy health check for D0")
		tags = append(tags, tagInfo{
			device:  "D",
			offset:  0,
			dbID:    "dummy",
			tagName: "dummy",
		})
	}

	go func() {
		var pool *ConnectionPool
		var err error
		activePort := plc.Port

		for _, p := range fallbackPorts {
			rpc, loopErr := newPlcConn(plc.IpAddress, p)
			if loopErr == nil {
				if p != plc.Port {
					log.Printf("[Poller] initial connect on fallback port %d", p)
					plc.Port = p
					activePort = p
					postgres.DB.Model(&plc).Update("port", p)
				}
				rpc.Close()
				err = nil
				break
			}
			err = loopErr
		}
		if err != nil {
			log.Printf("[Poller] initial connect error on all ports: %v", err)
		}
		
		pool, err = NewConnectionPool(plc.IpAddress, activePort, 10)
		if err != nil {
			log.Printf("[Poller] failed to init connection pool: %v", err)
		}

		var backoff time.Duration = 50 * time.Millisecond
		lastPublished := make(map[string]interface{})
		lastDbValues := make(map[string]int)
		const maxBackoff = 5 * time.Second

		// Group tags by device type and create dense read blocks to avoid polling large gaps
		type readBlock struct {
			device string
			start  int
			size   int
		}
		var blocks []readBlock

		deviceMap := make(map[string][]tagInfo)
		for _, t := range tags {
			deviceMap[t.device] = append(deviceMap[t.device], t)
		}

		for dev, devTags := range deviceMap {
			// Sort tags by offset
			sort.Slice(devTags, func(i, j int) bool {
				return devTags[i].offset < devTags[j].offset
			})

			isWord := (dev == "D" || dev == "W" || dev == "R" || dev == "ZR")
			maxChunk := 7168
			if isWord {
				maxChunk = 960
			}
			maxGap := 256 // Allow reading up to 256 unused registers if it saves a network request

			startOff := devTags[0].offset
			lastOff := devTags[0].offset

			for _, t := range devTags {
				if t.offset-startOff >= maxChunk || t.offset-lastOff > maxGap {
					// End current block and start a new one
					blocks = append(blocks, readBlock{device: dev, start: startOff, size: lastOff - startOff + 1})
					startOff = t.offset
				}
				lastOff = t.offset
			}
			blocks = append(blocks, readBlock{device: dev, start: startOff, size: lastOff - startOff + 1})
		}

		natsSubscribed := false

		writeQueue := make(chan writeReq, 5000)

		// Write Worker Goroutine
		go func() {
			var batch []writeReq
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case req := <-writeQueue:
					batch = append(batch, req)
					if len(batch) >= 50 {
						processWriteBatch(pool, batch)
						batch = batch[:0]
					}
				case <-ticker.C:
					if len(batch) > 0 {
						processWriteBatch(pool, batch)
						batch = batch[:0]
					}
				}
			}
		}()

		for {
			if pool == nil {
				// Re-init pool
				pool, err = NewConnectionPool(plc.IpAddress, plc.Port, 10)
				if err != nil {
					backoff = time.Duration(math.Min(float64(backoff*2), float64(maxBackoff)))
					healthJSON, _ := json.Marshal(map[string]interface{}{
						"status":     "offline",
						"latency_ms": 0,
					})
					nats.GetConn().Publish("plc.health."+plc.ID, healthJSON)
					PlcHealthMap.Store(plc.ID, PLCHealth{Status: "offline", LatencyMS: 0, LastSeen: time.Now()})
					time.Sleep(backoff)
					continue
				}
				backoff = 50 * time.Millisecond
			}

			// NATS Write Subscriber (Setup once per connection)
			if !natsSubscribed && nats_client.GetConn() != nil {
				subName := fmt.Sprintf("gateway.write.%s", plc.IpAddress)
				nats_client.GetConn().Subscribe(subName, func(m *nats_go.Msg) {
					var req writeReq
					if err := json.Unmarshal(m.Data, &req); err == nil {
						log.Printf("[Poller] Queued NATS write for %s%d = %d", req.Device, req.Offset, req.Value)
						select {
						case writeQueue <- req:
						default:
							log.Printf("[Poller] Write queue full, dropping %s%d", req.Device, req.Offset)
						}
					} else {
						log.Printf("[Poller] Failed to parse NATS write payload: %v", string(m.Data))
					}
				})
				natsSubscribed = true
			}

			var result sync.Map
			t0 := time.Now()
			var wg sync.WaitGroup
			
			// Job queue for workers
			jobs := make(chan readBlock, len(blocks))
			for _, blk := range blocks {
				jobs <- blk
			}
			close(jobs)

			// Worker pool implementation
			numWorkers := 10
			if len(blocks) < numWorkers {
				numWorkers = len(blocks)
			}

			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					for blk := range jobs {
						conn, err := pool.Acquire()
						if err != nil {
							atomic.AddInt32(&plcStats.ErrorCount, 1)
							continue
						}

						var frame []byte
						var vals []int

						if blk.device == "D" || blk.device == "W" || blk.device == "R" || blk.device == "ZR" {
							frame = BuildReadWordFrame(blk.device, blk.start, blk.size)
							vals, err = conn.DoReadWords(frame, blk.size)
						} else {
							frame = buildReadFrame(blk.device, blk.start, blk.size)
							vals, err = conn.DoRead(frame)
						}

						pool.Release(conn)

						if err != nil {
							log.Printf("[Poller] chunk read error at %s:%d: %v", blk.device, blk.start, err)
							atomic.AddInt32(&plcStats.ErrorCount, 1)
							continue
						}

						for _, t := range deviceMap[blk.device] {
							if t.offset >= blk.start && t.offset < blk.start+blk.size {
								pos := t.offset - blk.start
								if pos >= 0 && pos < len(vals) {
									result.Store(t.dbID, vals[pos])
								}
							}
						}
					}
				}()
			}
			wg.Wait()
			rtt := time.Since(t0)
			atomic.AddUint64(&plcStats.ReadOps, 1)
			atomic.AddUint64(&plcStats.ReadMicros, uint64(rtt.Microseconds()))
			backoff = 50 * time.Millisecond
			latency := float64(rtt.Microseconds()) / 1000.0
			healthJSON, _ := json.Marshal(map[string]interface{}{
				"status":     "online",
				"latency_ms": latency,
			})
			nats.GetConn().Publish("plc.health."+plc.ID, healthJSON)
			PlcHealthMap.Store(plc.ID, PLCHealth{Status: "online", LatencyMS: latency, LastSeen: time.Now()})

			// --- COMMENTED OUT REDIS OVERRIDES FOR TESTING ---
			/*
				overridesMap, _ := RedisClient.HGetAll(Ctx, "gateway:overrides").Result()

				var clearOverrides []string
				for _, t := range tags {
					if ov, exists := overridesMap[t.dbID]; exists && ov != "" {
						if ovVal, err := strconv.Atoi(ov); err == nil {
							if plcVal, ok := result[t.dbID]; ok {
								if plcVal == ovVal {
									clearOverrides = append(clearOverrides, t.dbID)
								} else {
									result[t.dbID] = ovVal
								}
							} else {
								result[t.dbID] = ovVal
							}
						}
					}
				}

				if len(clearOverrides) > 0 {
					RedisClient.HDel(Ctx, "gateway:overrides", clearOverrides...)
				}
			*/

			// Extract values for async processing
			capturedResult := make(map[string]int)
			result.Range(func(key, value interface{}) bool {
				capturedResult[key.(string)] = value.(int)
				return true
			})

			// Synchronously calculate diffs to avoid data races and map corruption
			var dbUpdates []models.MitsubishiTagValue
			changedForNats := false
			regs := make(map[string]interface{})
			tagIds := make(map[string]string)

			for _, t := range tags {
				if v, ok := capturedResult[t.dbID]; ok {
					// DB diff
					if lastV, exists := lastDbValues[t.dbID]; !exists || lastV != v {
						dbUpdates = append(dbUpdates, models.MitsubishiTagValue{
							TagID:     t.dbID,
							TagName:   t.tagName,
							Value:     v,
							UpdatedAt: time.Now(),
						})
						lastDbValues[t.dbID] = v
					}

					// NATS diff
					key := t.device + strconv.Itoa(t.offset)
					regs[key] = v
					tagIds[key] = t.dbID

					if lastV, ok := lastPublished[key]; !ok || lastV != v {
						changedForNats = true
					}
				}
			}

			if changedForNats {
				lastPublished = regs
			}

			// Decouple DB + NATS operations into a background Goroutine!
			go func(plcID, ip string, updates []models.MitsubishiTagValue, shouldPublish bool, pubRegs map[string]interface{}, pubTagIds map[string]string) {
				if len(updates) > 0 {
					if err := postgres.DB.Clauses(clause.OnConflict{
						Columns:   []clause.Column{{Name: "tag_id"}},
						DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at", "tag_name"}),
					}).Create(&updates).Error; err != nil {
						log.Printf("[Poller] DB upsert error: %v", err)
					}
				}

				if !shouldPublish {
					return
				}

				payload := map[string]interface{}{
					"ip":         ip,
					"protocol":   "MC Protocol",
					"plc_status": true,
					"registers":  pubRegs,
					"tag_ids":    pubTagIds,
					"timestamp":  time.Now().Unix(),
				}

				jsonPayload, _ := json.Marshal(payload)

				tNats0 := time.Now()
				if nats_client.GetConn() != nil {
					nats_client.GetConn().Publish("plc.data."+plcID, jsonPayload)
				}
				natsRtt := time.Since(tNats0)

				atomic.AddUint64(&plcStats.PubCount, 1)
				atomic.AddUint64(&plcStats.NatsPubMicros, uint64(natsRtt.Microseconds()))
			}(plc.ID, plc.IpAddress, dbUpdates, changedForNats, regs, tagIds)

			pollInterval := 200 * time.Millisecond
			if val, err := strconv.Atoi(os.Getenv("POLL_INTERVAL_MS")); err == nil && val > 0 {
				pollInterval = time.Duration(val) * time.Millisecond
			}
			time.Sleep(pollInterval)
		}
	}()

	// stats loop unchanged
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			rOps := atomic.LoadUint64(&plcStats.ReadOps)
			wOps := atomic.LoadUint64(&plcStats.WriteOps)
			pOps := atomic.LoadUint64(&plcStats.PubCount)

			avgR, avgW, avgNats := 0.0, 0.0, 0.0
			if rOps > 0 {
				avgR = float64(atomic.LoadUint64(&plcStats.ReadMicros)) / float64(rOps) / 1000
			}
			if wOps > 0 {
				avgW = float64(atomic.LoadUint64(&plcStats.WriteMicros)) / float64(wOps) / 1000
			}
			if pOps > 0 {
				avgNats = float64(atomic.LoadUint64(&plcStats.NatsPubMicros)) / float64(pOps) / 1000
			}

			log.Printf("[Poller MON] R=%.2fms W=%.2fms Err=%d | PubLatencies: NATS=%.3fms",
				avgR, avgW, atomic.LoadInt32(&plcStats.ErrorCount), avgNats)

			// --- COMMENTED OUT REDIS STATS SET FOR TESTING ---
			/*
				statsMap := map[string]float64{
					"read_latency_ms":       math.Round(avgR*100) / 100,
					"write_latency_ms":      math.Round(avgW*100) / 100,
					"redis_pub_latency_ms":  math.Round(avgRedis*1000) / 1000,
					"nats_pub_latency_ms":   math.Round(avgNats*1000) / 1000,
					"read_ops":              float64(rOps),
					"write_ops":             float64(wOps),
					"pub_ops":               float64(pOps),
					"error_count":           float64(atomic.LoadInt32(&plcStats.ErrorCount)),
				}

				if jsonStats, err := json.Marshal(statsMap); err == nil {
					RedisClient.Set(Ctx, "gateway:poller_stats", jsonStats, 0)
				}
			*/
		}
	}()
}

type writeReq struct {
	Device string `json:"device"`
	Offset int    `json:"offset"`
	Value  int    `json:"value"`
}

func processWriteBatch(pool *ConnectionPool, batch []writeReq) {
	if pool == nil {
		log.Printf("[Poller Writer] Drop %d writes (Pool not initialized)", len(batch))
		return
	}

	for _, req := range batch {
		var writeErr error
		retries := 0
		for retries < 3 {
			rpc, err := pool.Acquire()
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				retries++
				continue
			}

			if req.Device == "D" || req.Device == "W" || req.Device == "R" {
				writeErr = rpc.WriteSingleWord(req.Device, req.Offset, req.Value)
			} else {
				writeErr = rpc.WriteSingleBit(req.Device, req.Offset, req.Value)
			}
			pool.Release(rpc)

			if writeErr != nil {
				log.Printf("[Poller Writer] NATS write failed for %s%d (retry %d): %v", req.Device, req.Offset, retries, writeErr)
				retries++
				time.Sleep(50 * time.Millisecond)
			} else {
				break
			}
		}
		if writeErr == nil {
			atomic.AddUint64(&plcStats.WriteOps, 1)
			atomic.AddUint64(&plcStats.WriteMicros, uint64(5000)) // placeholder until rtt measurement
		}
	}
}
