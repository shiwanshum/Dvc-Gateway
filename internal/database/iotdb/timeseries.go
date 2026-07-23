package iotdb

import (
	"encoding/json"
	"log"

	"github.com/apache/iotdb-client-go/client"
	"github.com/program-dg/dvc-gateway/internal/nats"
	nats_go "github.com/nats-io/nats.go"
)

var session client.Session

// InitIoTDB connects to Apache IoTDB
func InitIoTDB(host string, port string, user string, pass string) error {
	config := &client.Config{
		Host:     host,
		Port:     port,
		UserName: user,
		Password: pass,
	}
	session = client.NewSession(config)
	if err := session.Open(false, 0); err != nil {
		log.Printf("Failed to connect to IoTDB: %v", err)
		return err
	}
	log.Println("Connected to IoTDB Successfully!")
	return nil
}

// StartNATSLogConsumer listens to real-time PLC data on NATS and inserts it into IoTDB
func StartNATSLogConsumer() {
	nc := nats.GetConn()
	if nc == nil {
		log.Println("NATS not connected, cannot start IoTDB consumer")
		return
	}

	nc.Subscribe("plc.data.>", func(m *nats_go.Msg) {
		// Example NATS payload expected from Mitsubishi poller
		var tagData struct {
			PLCIP     string             `json:"plc_ip"`
			Timestamp int64              `json:"timestamp"`
			Values    map[string]float32 `json:"values"`
		}

		if err := json.Unmarshal(m.Data, &tagData); err != nil {
			log.Printf("Error decoding NATS message: %v", err)
			return
		}

		// Prepare IoTDB timeseries insertion (Device path: root.plc.<ip>)
		devicePath := "root.plc." + formatIP(tagData.PLCIP)
		var measurements []string
		var dataTypes []client.TSDataType
		var values []interface{}

		for tagName, val := range tagData.Values {
			measurements = append(measurements, tagName)
			dataTypes = append(dataTypes, client.FLOAT)
			values = append(values, val)
		}

		if len(measurements) > 0 {
			err := session.InsertRecord(devicePath, measurements, dataTypes, values, tagData.Timestamp)
			if err != nil {
				log.Printf("Failed to insert into IoTDB: %v", err)
			}
		}
	})
	log.Println("Started NATS to IoTDB logger consumer...")
}

func formatIP(ip string) string {
	// Replaces dots with underscores for IoTDB paths
	result := ""
	for _, c := range ip {
		if c == '.' {
			result += "_"
		} else {
			result += string(c)
		}
	}
	return result
}
