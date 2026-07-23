package handlers

import (
	"encoding/json"
	"log"

	"github.com/gofiber/websocket/v2"
	"github.com/program-dg/dvc-gateway/internal/nats"
	nats_go "github.com/nats-io/nats.go"
)

func WsHandler(c *websocket.Conn) {
	nc := nats.GetConn()
	if nc == nil {
		log.Println("NATS not connected, closing WS")
		return
	}

	sub, err := nc.Subscribe("plc.data.>", func(m *nats_go.Msg) {
		var tagData map[string]interface{}
		json.Unmarshal(m.Data, &tagData)
		if err := c.WriteJSON(tagData); err != nil {
			log.Printf("WS write failed: %v", err)
		}
	})
	
	if err != nil {
		log.Printf("NATS subscribe failed: %v", err)
		return
	}
	defer sub.Unsubscribe()

	// Keep connection alive
	var (
		msg []byte
	)
	for {
		if _, msg, err = c.ReadMessage(); err != nil {
			break
		}
		log.Println("WS message received:", string(msg))
	}
}
