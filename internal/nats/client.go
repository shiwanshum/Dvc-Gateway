package nats

import (
	"log"
	nats_go "github.com/nats-io/nats.go"
)

var (
	conn *nats_go.Conn
	js   nats_go.JetStreamContext
)

func InitNATS(url string) error {
	var err error
	conn, err = nats_go.Connect(url)
	if err != nil {
		log.Printf("Failed to connect to NATS at %s: %v", url, err)
		return err
	}
	
	// Initialize JetStream Context
	js, err = conn.JetStream()
	if err != nil {
		log.Printf("Failed to initialize JetStream: %v", err)
		return err
	}

	log.Printf("Connected to NATS & JetStream at %s", url)
	return nil
}

func GetConn() *nats_go.Conn {
	return conn
}

func GetJS() nats_go.JetStreamContext {
	return js
}
