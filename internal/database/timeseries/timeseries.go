package timeseries

import (
	"encoding/json"
	"log"
	"time"

	"github.com/program-dg/dvc-gateway/internal/database/postgres"
	"github.com/program-dg/dvc-gateway/internal/nats"
	nats_go "github.com/nats-io/nats.go"
)

// StartNATSLogConsumer listens to real-time PLC data on NATS and inserts it into TimescaleDB
func StartNATSLogConsumer() {
	nc := nats.GetConn()
	if nc == nil {
		log.Println("NATS not connected, cannot start Timeseries consumer")
		return
	}

	nc.Subscribe("plc.data.>", func(m *nats_go.Msg) {
		var tagData struct {
			PLCIP     string             `json:"plc_ip"`
			Timestamp int64              `json:"timestamp"`
			Values    map[string]float32 `json:"values"`
		}

		if err := json.Unmarshal(m.Data, &tagData); err != nil {
			log.Printf("Error decoding NATS message: %v", err)
			return
		}

		// Convert timestamp (assuming it's in milliseconds based on common IoT standards)
		ts := time.UnixMilli(tagData.Timestamp)

		// Insert each tag value into the telemetry.plc_data hypertable
		for tagName, val := range tagData.Values {
			err := postgres.DB.Exec(
				"INSERT INTO telemetry.plc_data (time, plc_ip, tag_name, value) VALUES (?, ?, ?, ?)",
				ts, tagData.PLCIP, tagName, val,
			).Error

			if err != nil {
				log.Printf("Failed to insert telemetry data into TimescaleDB: %v", err)
			}
		}
	})
	log.Println("Started NATS to TimescaleDB logger consumer...")
}
