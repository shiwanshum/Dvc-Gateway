package main

import (
	"fmt"
	"net"
	"time"
)

func main() {
	for i := 0; i < 5; i++ {
		conn, err := net.Dial("tcp", "192.169.4.152:1026")
		if err != nil {
			fmt.Printf("Attempt %d: Error %v\n", i, err)
		} else {
			fmt.Printf("Attempt %d: Success\n", i)
			conn.Close()
		}
		time.Sleep(1 * time.Second)
	}
}
