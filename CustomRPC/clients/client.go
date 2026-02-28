package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	"sync/atomic"

	"github.com/shubhdevelop/proto-playground/customRPC/rpc"
)

var SeqCounter uint32 = 0

type AddArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

func main() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	makeNCalls(10, &conn)
}

func makeNCalls(n int, conn *net.Conn) {
	for range n {
		// Build request
		req := map[string]any{
			"method": "Arithmetic.Add",
			"params": AddArgs{
				A: rand.Intn(10),
				B: rand.Intn(10),
			},
		}

		payload, _ := json.Marshal(req)

		// Wrap with headerseqID := atomic.AddUint32(&seqCounter, 1)
		seqID := atomic.AddUint32(&SeqCounter, 1)
		msg := rpc.NewMessage(seqID, payload)

		// Send
		//
		(*conn).Write(msg)

		// Read response
		_, respPayload, err := rpc.ReadMessage((*conn))
		if err != nil {
			panic(err)
		}

		fmt.Println("Response:", string(respPayload))

	}
}
