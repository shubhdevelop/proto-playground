package main

import (
	"context"
	"fmt"
	"log"
	"time"

	commands "github.com/shubhdevelop/proto-playground/valkey-proto/internals"
)

func main() {
	time.Sleep(30 * time.Second)
	ctx := context.Background()
	fmt.Println("Valkey Proto playground")

	// ✅ FIXED: Handle the error
	valkey, err := commands.NewValkey(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Valkey: %v", err)
	}
	defer valkey.Client.Close()

	val := valkey.GET(ctx, "name")
	fmt.Println("GET name:", val)

	val = valkey.Set(ctx, "name", "shubham")
	fmt.Println("SET name:", val)

	val = valkey.GET(ctx, "name")
	fmt.Println("GET name:", val)

	val = valkey.SetNx(ctx, "name2", "yash")
	fmt.Println("SETNX name2:", val)

	val = valkey.GET(ctx, "name2")
	fmt.Println("GET name2:", val)

	val = valkey.SetNx(ctx, "name2", "yash")
	fmt.Println("SETNX name2 (should fail):", val)

	val = valkey.GET(ctx, "name2")
	fmt.Println("GET name2:", val)
}
