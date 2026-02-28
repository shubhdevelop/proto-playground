// Package customrpc
package rpc

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
)

var (
	MagicNumber uint32 = 0xCAFEBABE
	SeqCounter  uint32 = 0
)

func NewMessage(SeqID uint32, payload []byte) []byte {
	payloadLen := len(payload)
	// Allocate upfront (12 bytes header + payload)
	message := make([]byte, 12+payloadLen)
	binary.BigEndian.PutUint32(message[0:4], MagicNumber)
	binary.BigEndian.PutUint32(message[4:8], SeqID)
	binary.BigEndian.PutUint32(message[8:12], uint32(payloadLen))

	copy(message[12:], payload)
	return message
}

func ReadMessage(conn io.Reader) (uint32, []byte, error) {
	header := make([]byte, 12)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return 0, nil, fmt.Errorf("read header failed: %w", err)
	}

	magic := binary.BigEndian.Uint32(header[0:4])
	if magic != MagicNumber {
		return 0, nil, errors.New("invalid magic number")
	}

	seqID := binary.BigEndian.Uint32(header[4:8])
	payloadLen := binary.BigEndian.Uint32(header[8:12])

	payload := make([]byte, payloadLen)

	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return 0, nil, fmt.Errorf("read payload failed: %w", err)
	}

	return seqID, payload, nil
}

type MethodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
	receiver  reflect.Value
}

// Create a new instance of the argument type
func (m *MethodType) newArgv() reflect.Value {
	return reflect.New(m.ArgType.Elem())
}

// Create a new instance of the reply type (always a pointer)
func (m *MethodType) newReplyv() reflect.Value {
	replyv := reflect.New(m.ReplyType.Elem())
	return replyv
}

type Server struct {
	services map[string]*MethodType
}

func NewServer() *Server {
	return &Server{
		services: make(map[string]*MethodType),
	}
}

func (s *Server) Start(address string, codec Codec) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	fmt.Println("RPC Server listening on", address)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Accept error:", err)
			continue
		}

		go s.HandleConnection(conn, codec)
	}
}

func (s *Server) Register(receiver any) error {
	typ := reflect.TypeOf(receiver)
	val := reflect.ValueOf(receiver)

	serviceName := reflect.Indirect(val).Type().Name()

	for i := 0; i < typ.NumMethod(); i++ {
		method := typ.Method(i)
		mType := method.Type

		if mType.NumIn() != 3 ||
			mType.NumOut() != 1 ||
			mType.Out(0) != reflect.TypeFor[error]() {
			continue
		}

		if mType.In(2).Kind() != reflect.Pointer {
			continue
		}

		methodName := method.Name
		key := serviceName + "." + methodName

		s.services[key] = &MethodType{
			method:    method,
			ArgType:   mType.In(1),
			receiver:  val,
			ReplyType: mType.In(2),
		}
	}

	return nil
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type Codec interface {
	Encode(any) ([]byte, error)
	Decode([]byte, any) error
	Type() string // e.g., "application/x-protobuf"
}

type JSONCodec struct {
	Codec
}

func (j *JSONCodec) Encode(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (j *JSONCodec) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (j *JSONCodec) Type() string {
	return "json"
}

func (s *Server) HandleConnection(conn io.ReadWriteCloser, codec Codec) {
	defer conn.Close()

	for {
		seqID, payload, err := ReadMessage(conn)
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Println("Client disconnected")
			} else {
				fmt.Println("Read error:", err)
			}
			return
		}

		// Step 1: Decode Request wrapper
		var req Request
		if err := codec.Decode(payload, &req); err != nil {
			fmt.Println("Decode request error:", err)
			continue
		}

		// Step 2: Find method
		m, ok := s.services[req.Method]
		if !ok {
			fmt.Println("Method not found:", req.Method)
			continue
		}

		// Step 3: Create argument
		argv := m.newArgv()
		if err := json.Unmarshal(req.Params, argv.Interface()); err != nil {
			fmt.Println("Param decode error:", err)
			continue
		}

		// Step 4: Create reply
		replyv := m.newReplyv()

		// Step 5: Call method
		returnValues := m.method.Func.Call([]reflect.Value{
			m.receiver,
			argv,
			replyv,
		})

		if errInter := returnValues[0].Interface(); errInter != nil {
			fmt.Println("Method error:", errInter)
			continue
		}

		// Step 6: Send response
		respPayload, _ := codec.Encode(replyv.Interface())
		respMsg := NewMessage(seqID, respPayload)
		conn.Write(respMsg)

		fmt.Println("Handled request with SeqID:", seqID)
	}
}
