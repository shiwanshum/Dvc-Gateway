package mitsubishi

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

var deviceCode = map[string]byte{
	"X": 0x9C, "Y": 0x9D, "M": 0x90, "L": 0x92,
	"F": 0x93, "V": 0x94, "B": 0xA0, "W": 0xB4, "D": 0xA8,
	"R": 0xAF, "ZR": 0xB0,
}

const ioTimeout = 2000 * time.Millisecond

type PlcConn struct {
	host      string
	port      int    // Read port
	writePort int    // Write port (0 = same as port)
	conn      *net.TCPConn
	recvBuf   [65536]byte
	mu        sync.Mutex
}

func newPlcConn(host string, port int) (*PlcConn, error) {
	return newPlcConnWithWritePort(host, port, 0)
}

func newPlcConnWithWritePort(host string, port, writePort int) (*PlcConn, error) {
	pc := &PlcConn{host: host, port: port, writePort: writePort}
	if err := pc.dial(); err != nil {
		return nil, err
	}
	return pc, nil
}

func (pc *PlcConn) dial() error {
	if pc.conn != nil {
		pc.conn.SetLinger(0)
		pc.conn.Close()
		pc.conn = nil
	}
	addr := net.JoinHostPort(pc.host, strconv.Itoa(pc.port))
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}
	tcp := conn.(*net.TCPConn)
	tcp.SetNoDelay(true)
	tcp.SetLinger(0)
	tcp.SetKeepAlive(true)
	tcp.SetKeepAlivePeriod(10 * time.Second)
	tcp.SetReadBuffer(2 * 1024 * 1024)
	tcp.SetWriteBuffer(2 * 1024 * 1024)
	pc.conn = tcp
	return nil
}

func (pc *PlcConn) send(frame []byte) error {
	_, err := pc.conn.Write(frame)
	return err
}

func (pc *PlcConn) recv(n int) error {
	_, err := io.ReadFull(pc.conn, pc.recvBuf[:n])
	return err
}

func (pc *PlcConn) Close() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	if pc.conn != nil {
		pc.conn.SetLinger(0)
		pc.conn.Close()
		pc.conn = nil
	}
}

func buildReadFrame(device string, offset, count int) []byte {
	return []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		0x0C, 0x00, // Data len (12 bytes)
		0x10, 0x00, // CPU Timer
		0x01, 0x04, 0x01, 0x00, // Cmd & Subcmd (Bit batch read)
		byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device], // Offset & Code
		byte(count), byte(count >> 8), // Count
	}
}

func buildWriteFrame(device string, offset, count int, toggle bool) []byte {
	code := deviceCode[device]
	off4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(off4, uint32(offset))
	off3 := off4[:3]
	cnt2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(cnt2, uint16(count))

	payloadLen := (count + 1) / 2

	wbody := []byte{0x01, 0x14, 0x01, 0x00, off3[0], off3[1], off3[2], code, cnt2[0], cnt2[1]}
	var wdata []byte
	if toggle {
		wdata = make([]byte, payloadLen)
		full := count / 2
		for i := range full {
			wdata[i] = 0x11
		}
		if count%2 == 1 {
			wdata[full] = 0x10
		}
	} else {
		wdata = make([]byte, payloadLen)
	}

	wtotal := append([]byte{0x10, 0x00}, wbody...)
	wtotal = append(wtotal, wdata...)
	wdl := make([]byte, 2)
	binary.LittleEndian.PutUint16(wdl, uint16(len(wtotal)))
	frame := append([]byte{0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00}, wdl...)
	frame = append(frame, wtotal...)
	return frame
}

func buildWriteFrames(device string, startOffset, totalCount, chunkSize int, toggle bool) [][]byte {
	var frames [][]byte
	end := startOffset + totalCount
	for offset := startOffset; offset < end; offset += chunkSize {
		count := chunkSize
		if offset+count > end {
			count = end - offset
		}
		frames = append(frames, buildWriteFrame(device, offset, count, toggle))
	}
	return frames
}

func (pc *PlcConn) readBitValues(frame []byte) ([]int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doRead(frame)
}

// doRead sends a read frame and returns parsed bit values (no mutex)
func (pc *PlcConn) doRead(frame []byte) ([]int, error) {
	if pc.conn == nil {
		return nil, fmt.Errorf("not connected")
	}
	if err := pc.send(frame); err != nil {
		pc.Close()
		return nil, fmt.Errorf("send: %w", err)
	}
	if err := pc.recv(11); err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			if retryErr := pc.recv(11); retryErr == nil {
				goto checkEC
			}
		}
		pc.Close()
		return nil, fmt.Errorf("recv: %w", err)
	}
checkEC:
	respLen := int(pc.recvBuf[7]) | int(pc.recvBuf[8])<<8
	remaining := respLen - 2
	if remaining > 0 && remaining < 32768 {
		if _, err := io.ReadFull(pc.conn, pc.recvBuf[11:11+remaining]); err != nil {
			pc.Close()
			return nil, fmt.Errorf("recv data: %w", err)
		}
	}

	ec := int(pc.recvBuf[9]) | int(pc.recvBuf[10])<<8
	
	// DRAIN STRAY BYTES removed to eliminate 10ms artificial latency.
	// Stray bytes indicate a protocol framing error, which will correctly 
	// fail the next recv(11) and reset the connection.

	if ec != 0 {
		return nil, fmt.Errorf("PLC fault 0x%04X", ec)
	}

	if remaining <= 0 || remaining >= 32768 {
		return nil, fmt.Errorf("invalid response length %d", respLen)
	}
	count := remaining * 2
	vals := make([]int, count)
	for i := range count {
		b := pc.recvBuf[11 + i/2]
		if i%2 == 0 {
			if b&0xF0 != 0 {
				vals[i] = 1
			}
		} else {
			if b&0x0F != 0 {
				vals[i] = 1
			}
		}
	}
	return vals, nil
}

var scanPorts = []int{1025, 1026, 1024, 2000, 5000, 5001, 5002, 5003, 5004, 5005, 5006, 5007, 5010, 3000, 4000, 5555, 6000}

// findMCPort scans common ports on host for an MC Protocol responder.
// Tries preferredPort first; if it fails, scans scanPorts list.
// Returns the port that responds to an MC read frame, or error.
func findMCPort(host string, preferredPort int) (int, error) {
	// Try preferred port first
	if preferredPort > 0 {
		if ProbeMCPort(host, preferredPort) {
			return preferredPort, nil
		}
	}
	// Scan other ports
	for _, p := range scanPorts {
		if p == preferredPort {
			continue
		}
		if ProbeMCPort(host, p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("no MC protocol responder found on %s (scanned %v)", host, scanPorts)
}

// ScanAllMCPorts scans all known common ports concurrently and returns a list of open, responding ports.
func ScanAllMCPorts(host string) []int {
	var openPorts []int
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, p := range scanPorts {
		wg.Add(1)
		go func(port int) {
			defer wg.Done()
			if ProbeMCPort(host, port) {
				mu.Lock()
				openPorts = append(openPorts, port)
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()
	return openPorts
}

// ProbeMCPort tries a TCP connection and sends an MC read frame for M0, 1 point.
// Returns true if the port responds with a valid MC protocol response (no fault).
func ProbeMCPort(host string, port int) bool {
	addr := net.JoinHostPort(host, strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(3 * time.Second))

	// Build MC read frame for M0, 1 point
	frame := buildReadFrame("M", 0, 1)
	if _, err := conn.Write(frame); err != nil {
		return false
	}

	resp := make([]byte, 11)
	if _, err := io.ReadFull(conn, resp); err != nil {
		return false
	}
	// Check complete code at bytes 9-10 (0x0000 = no error)
	if len(resp) >= 11 {
		ec := int(resp[9])<<8 | int(resp[10])
		return ec == 0
	}
	return false
}

// NewPlcConn creates a new PLC connection to the given host:port.
func NewPlcConn(host string, port int) (*PlcConn, error) {
	return newPlcConn(host, port)
}

// WriteSingleBit sends a single-bit write via MC Protocol.
func (pc *PlcConn) WriteSingleBit(device string, offset, value int) error {
	return pc.writeSingleBit(device, offset, value)
}

// WriteSingleWord sends a single-word (16-bit) write via MC Protocol.
func (pc *PlcConn) WriteSingleWord(device string, offset, value int) error {
	// Zero-allocation pre-computed 3E frame (23 bytes total)
	frame := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		0x0E, 0x00, // Data length (14 bytes)
		0x10, 0x00, // CPU monitoring timer
		0x01, 0x14, 0x00, 0x00, // Cmd & Subcmd (Word batch write)
		byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device], // Offset & Code
		0x01, 0x00, // Count = 1
		byte(value), byte(value >> 8), // 16-bit Value
	}

	if pc.writePort != 0 && pc.writePort != pc.port {
		wc, err := newPlcConn(pc.host, pc.writePort)
		if err != nil {
			return fmt.Errorf("write conn: %w", err)
		}
		defer wc.Close()
		wc.mu.Lock()
		defer wc.mu.Unlock()
		return wc.doWrite(frame)
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doWrite(frame)
}

// writeSingleBit sends a single-bit write frame via MC Protocol 3E frame (bit batch write).
// If writePort differs from read port, creates a temporary connection for the write.
func (pc *PlcConn) writeSingleBit(device string, offset int, value int) error {
	bitVal := byte(0x00)
	if value != 0 {
		bitVal = 0x10
	}

	frame := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		0x0D, 0x00, // Data len (13 bytes)
		0x10, 0x00, // CPU Timer
		0x01, 0x14, 0x01, 0x00, // Cmd & Subcmd (Bit batch write)
		byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device], // Offset & Code
		0x01, 0x00, // Count = 1
		bitVal, // 1-bit Value
	}

	if pc.writePort != 0 && pc.writePort != pc.port {
		wc, err := newPlcConn(pc.host, pc.writePort)
		if err != nil {
			return fmt.Errorf("write conn: %w", err)
		}
		defer wc.Close()
		wc.mu.Lock()
		defer wc.mu.Unlock()
		return wc.doWrite(frame)
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doWrite(frame)
}

// BuildReadFrame builds an MC read frame (exported for testing).
func BuildReadFrame(device string, offset, count int) []byte {
	return buildReadFrame(device, offset, count)
}

// BuildReadWordFrame builds an MC read frame for Word units.
func BuildReadWordFrame(device string, offset, count int) []byte {
	return []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		0x0C, 0x00, // Data len (12 bytes)
		0x10, 0x00, // CPU Timer
		0x01, 0x04, 0x00, 0x00, // Cmd & Subcmd (Word batch read)
		byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device], // Offset & Code
		byte(count), byte(count >> 8), // Count
	}
}

// DoReadWords sends a word read frame and returns parsed integer values.
func (pc *PlcConn) DoReadWords(frame []byte, count int) ([]int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if pc.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	if err := pc.send(frame); err != nil {
		pc.Close()
		return nil, fmt.Errorf("send read: %w", err)
	}

	pc.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := pc.conn.Read(pc.recvBuf[:])
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			n, err = pc.conn.Read(pc.recvBuf[:])
			if err != nil {
				pc.Close()
				return nil, fmt.Errorf("retry recv: %w", err)
			}
		} else {
			pc.Close()
			return nil, fmt.Errorf("recv: %w", err)
		}
	}

	if n < 11 {
		if _, err := io.ReadFull(pc.conn, pc.recvBuf[n:11]); err != nil {
			pc.Close()
			return nil, fmt.Errorf("recv header: %w", err)
		}
		n = 11
	}

	respLen := int(pc.recvBuf[7]) | int(pc.recvBuf[8])<<8
	remaining := respLen - 2
	totalExpected := 11 + remaining

	if totalExpected > len(pc.recvBuf) {
		return nil, fmt.Errorf("packet too large: %d", totalExpected)
	}

	if n < totalExpected {
		if _, err := io.ReadFull(pc.conn, pc.recvBuf[n:totalExpected]); err != nil {
			pc.Close()
			return nil, fmt.Errorf("recv data: %w", err)
		}
	}

	ec := int(pc.recvBuf[9]) | int(pc.recvBuf[10])<<8

	// DRAIN STRAY BYTES removed to eliminate 10ms artificial latency.

	if ec != 0 {
		return nil, fmt.Errorf("PLC fault 0x%04X", ec)
	}

	results := make([]int, 0, count)
	for i := 0; i < count; i++ {
		idx := 11 + i*2
		if idx+2 > 11+remaining {
			break
		}
		val := int(binary.LittleEndian.Uint16(pc.recvBuf[idx : idx+2]))
		results = append(results, val)
	}
	return results, nil
}

// DoRead sends a read frame and returns parsed bit values (exported for testing).
func (pc *PlcConn) DoRead(frame []byte) ([]int, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doRead(frame)
}

// WriteBatchBits writes an array of bit values to the PLC in batch chunks.
func (pc *PlcConn) WriteBatchBits(device string, startOffset int, values []int) error {
	const maxChunk = 7168 // MC Protocol max bit write
	n := len(values)
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for off := 0; off < n; off += maxChunk {
		end := off + maxChunk
		if end > n {
			end = n
		}
		chunk := values[off:end]
		frame := buildWriteFrameCustom(device, startOffset+off, chunk)
		if err := pc.doWrite(frame); err != nil {
			return fmt.Errorf("batch write at offset %d: %w", startOffset+off, err)
		}
	}
	return nil
}

// WriteBatchWords writes an array of word values to the PLC in batch chunks.
func (pc *PlcConn) WriteBatchWords(device string, startOffset int, values []int) error {
	const maxChunk = 960 // MC Protocol max word write
	n := len(values)
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for off := 0; off < n; off += maxChunk {
		end := off + maxChunk
		if end > n {
			end = n
		}
		chunk := values[off:end]
		frame := buildWriteWordFrameCustom(device, startOffset+off, chunk)
		if err := pc.doWrite(frame); err != nil {
			return fmt.Errorf("batch word write at offset %d: %w", startOffset+off, err)
		}
	}
	return nil
}

// buildWriteWordFrameCustom builds a batch word-write frame with custom word values.
func buildWriteWordFrameCustom(device string, offset int, values []int) []byte {
	code := deviceCode[device]
	count := len(values)
	payloadLen := count * 2

	wdata := make([]byte, payloadLen)
	for i, v := range values {
		binary.LittleEndian.PutUint16(wdata[i*2:i*2+2], uint16(v))
	}

	off4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(off4, uint32(offset))
	off3 := off4[:3]
	cnt2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(cnt2, uint16(count))

	// Data length = timer(2) + cmd(2) + subcmd(2) + offset(3) + code(1) + count(2) + payloadLen
	dataLen := 2 + 2 + 2 + 3 + 1 + 2 + payloadLen
	dl2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(dl2, uint16(dataLen))

	frame := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		dl2[0], dl2[1], // Data Len
		0x10, 0x00, // CPU Timer
		0x01, 0x14, 0x00, 0x00, // Cmd & Subcmd (Word batch write)
		off3[0], off3[1], off3[2], code, // Offset & Device code
		cnt2[0], cnt2[1], // Count
	}
	frame = append(frame, wdata...)
	return frame
}

// buildWriteFrameCustom builds a batch bit-write frame with custom bit values.
func buildWriteFrameCustom(device string, offset int, values []int) []byte {
	code := deviceCode[device]
	count := len(values)
	payloadLen := (count + 1) / 2

	wdata := make([]byte, payloadLen)
	for i, v := range values {
		byteIdx := i / 2
		var nibbleVal byte = 0
		if v != 0 {
			nibbleVal = 1
		}
		if i%2 == 0 {
			wdata[byteIdx] |= nibbleVal << 4
		} else {
			wdata[byteIdx] |= nibbleVal
		}
	}

	off4 := make([]byte, 4)
	binary.LittleEndian.PutUint32(off4, uint32(offset))
	off3 := off4[:3]
	cnt2 := make([]byte, 2)
	binary.LittleEndian.PutUint16(cnt2, uint16(count))

	wbody := []byte{0x01, 0x14, 0x01, 0x00, off3[0], off3[1], off3[2], code, cnt2[0], cnt2[1]}
	wtotal := append([]byte{0x10, 0x00}, wbody...)
	wtotal = append(wtotal, wdata...)
	wdl := make([]byte, 2)
	binary.LittleEndian.PutUint16(wdl, uint16(len(wtotal)))
	frame := append([]byte{0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00}, wdl...)
	frame = append(frame, wtotal...)
	return frame
}

// doWrite sends a write frame and waits for acknowledgment (no mutex)
func (pc *PlcConn) doWrite(frame []byte) error {
	if pc.conn == nil {
		return fmt.Errorf("not connected")
	}
	if err := pc.send(frame); err != nil {
		pc.Close()
		return fmt.Errorf("send: %w", err)
	}
	
	pc.conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := pc.conn.Read(pc.recvBuf[:])
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			n, err = pc.conn.Read(pc.recvBuf[:])
			if err != nil {
				pc.Close()
				return fmt.Errorf("retry recv: %w", err)
			}
		} else {
			pc.Close()
			return fmt.Errorf("recv: %w", err)
		}
	}

	if n < 11 {
		if _, err := io.ReadFull(pc.conn, pc.recvBuf[n:11]); err != nil {
			pc.Close()
			return fmt.Errorf("recv header: %w", err)
		}
		n = 11
	}

	respLen := int(pc.recvBuf[7]) | int(pc.recvBuf[8])<<8
	remaining := respLen - 2
	totalExpected := 11 + remaining

	if totalExpected > len(pc.recvBuf) {
		return fmt.Errorf("packet too large: %d", totalExpected)
	}

	if n < totalExpected {
		if _, err := io.ReadFull(pc.conn, pc.recvBuf[n:totalExpected]); err != nil {
			pc.Close()
			return fmt.Errorf("recv data: %w", err)
		}
	}

	ec := int(pc.recvBuf[9]) | int(pc.recvBuf[10])<<8

	// DRAIN STRAY BYTES removed to eliminate 10ms artificial latency.

	if ec != 0 {
		return fmt.Errorf("PLC fault 0x%04X", ec)
	}
	return nil
}

// WriteFloat32 sends a 32-bit float write (takes 2 Words).
func (pc *PlcConn) WriteFloat32(device string, offset int, value float32) error {
	bits := math.Float32bits(value)
	low := int(bits & 0xFFFF)
	high := int(bits >> 16)
	
	frame := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
		0x10, 0x00, // Data length (16 bytes)
		0x10, 0x00, // CPU timer
		0x01, 0x14, 0x00, 0x00, // Cmd & Subcmd (Word batch write)
		byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device], // Offset & Code
		0x02, 0x00, // Count = 2 words
		byte(low), byte(low >> 8), // Word 1 (low)
		byte(high), byte(high >> 8), // Word 2 (high)
	}

	if pc.writePort != 0 && pc.writePort != pc.port {
		wc, err := newPlcConn(pc.host, pc.writePort)
		if err != nil {
			return fmt.Errorf("write conn: %w", err)
		}
		defer wc.Close()
		wc.mu.Lock()
		defer wc.mu.Unlock()
		return wc.doWrite(frame)
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doWrite(frame)
}

// WriteString sends a string write (each word = 2 characters, padded with null if odd length).
func (pc *PlcConn) WriteString(device string, offset int, value string) error {
	bytesVal := []byte(value)
	if len(bytesVal)%2 != 0 {
		bytesVal = append(bytesVal, 0x00) // pad to even length
	}
	wordCount := len(bytesVal) / 2
	
	frame := []byte{
		0x50, 0x00, 0x00, 0xFF, 0xFF, 0x03, 0x00, // Header
	}
	
	reqLen := 12 + len(bytesVal)
	frame = append(frame, byte(reqLen), byte(reqLen>>8))
	frame = append(frame, 0x10, 0x00) // timer
	frame = append(frame, 0x01, 0x14, 0x00, 0x00) // cmd/subcmd
	frame = append(frame, byte(offset), byte(offset >> 8), byte(offset >> 16), deviceCode[device])
	frame = append(frame, byte(wordCount), byte(wordCount>>8))
	frame = append(frame, bytesVal...)

	if pc.writePort != 0 && pc.writePort != pc.port {
		wc, err := newPlcConn(pc.host, pc.writePort)
		if err != nil {
			return fmt.Errorf("write conn: %w", err)
		}
		defer wc.Close()
		wc.mu.Lock()
		defer wc.mu.Unlock()
		return wc.doWrite(frame)
	}

	pc.mu.Lock()
	defer pc.mu.Unlock()
	return pc.doWrite(frame)
}

// DoReadFloat32 reads 2 words and converts to a 32-bit float.
func (pc *PlcConn) DoReadFloat32(device string, offset int) (float32, error) {
	frame := BuildReadWordFrame(device, offset, 2)
	words, err := pc.DoReadWords(frame, 2)
	if err != nil {
		return 0, err
	}
	if len(words) < 2 {
		return 0, fmt.Errorf("insufficient data for float32")
	}
	bits := uint32(words[0]) | (uint32(words[1]) << 16)
	return math.Float32frombits(bits), nil
}

// DoReadString reads n words and converts to string.
func (pc *PlcConn) DoReadString(device string, offset int, wordCount int) (string, error) {
	frame := BuildReadWordFrame(device, offset, wordCount)
	words, err := pc.DoReadWords(frame, wordCount)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, w := range words {
		sb.WriteByte(byte(w & 0xFF))
		sb.WriteByte(byte(w >> 8))
	}
	return strings.TrimRight(sb.String(), "\x00"), nil
}

// ConnectionPool manages multiple connections to the same PLC.
type ConnectionPool struct {
	host       string
	port       int
	conns      chan *PlcConn
	poolSize   int
	mu         sync.Mutex
	isClosed   bool
}

// NewConnectionPool initializes a pool of PLC connections.
func NewConnectionPool(host string, port int, size int) (*ConnectionPool, error) {
	pool := &ConnectionPool{
		host:     host,
		port:     port,
		conns:    make(chan *PlcConn, size),
		poolSize: size,
	}

	for i := 0; i < size; i++ {
		conn, err := newPlcConn(host, port)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("failed to create pool connection: %w", err)
		}
		pool.conns <- conn
	}

	return pool, nil
}

// Acquire gets a connection from the pool. It blocks until one is available.
func (cp *ConnectionPool) Acquire() (*PlcConn, error) {
	cp.mu.Lock()
	if cp.isClosed {
		cp.mu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}
	cp.mu.Unlock()

	conn := <-cp.conns
	return conn, nil
}

// Release returns a connection to the pool.
func (cp *ConnectionPool) Release(conn *PlcConn) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.isClosed {
		conn.Close()
		return
	}

	// Simple check: if connection is dead, try to reconnect
	if conn.conn == nil {
		_ = conn.dial()
	}
	
	select {
	case cp.conns <- conn:
	default:
		conn.Close() // Pool full, drop connection
	}
}

// Close closes all connections in the pool.
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.isClosed {
		return
	}
	cp.isClosed = true
	close(cp.conns)
	for conn := range cp.conns {
		conn.Close()
	}
}

