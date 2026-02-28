package main

import "github.com/shubhdevelop/proto-playground/customRPC/rpc"

type Arithmetic struct{}

type AddArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

type AddReply struct {
	Result int `json:"result"`
}

func (a *Arithmetic) Add(args AddArgs, reply *AddReply) error {
	reply.Result = args.A + args.B
	return nil
}

func main() {
	server := rpc.NewServer()
	server.Register(&Arithmetic{})

	codec := &rpc.JSONCodec{}

	server.Start(":8080", codec)
}
