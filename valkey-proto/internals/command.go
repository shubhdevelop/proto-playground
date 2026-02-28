package commands

import (
	"context"
	"fmt"
	"log"
	"time"

	glide "github.com/valkey-io/valkey-glide/go/v2"
	"github.com/valkey-io/valkey-glide/go/v2/config"
	"github.com/valkey-io/valkey-glide/go/v2/pipeline"
)

type ValkeyCommand struct {
	Client *glide.ClusterClient
}

// ✅ FIXED: Return error instead of ignoring it
func NewValkey(ctx context.Context) (*ValkeyCommand, error) {
	client, err := ValkeyInit(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Valkey: %w", err)
	}
	return &ValkeyCommand{
		Client: client,
	}, nil
}

func ValkeyInit(ctx context.Context) (*glide.ClusterClient, error) {
	log.Println("Connecting to Valkey cluster...")

	clientConfig := config.NewClusterClientConfiguration().
		WithAddress(&config.NodeAddress{Host: "valkey0", Port: 6379}).
		WithRequestTimeout(10 * time.Second).WithReadFrom(config.PreferReplica) // Optional: read from replicas

	log.Println("Creating cluster client...")
	client, err := glide.NewClusterClient(clientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Valkey cluster client: %w", err)
	}

	log.Println("Pinging cluster...")
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	res, err := client.Ping(pingCtx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to ping Valkey cluster: %w", err)
	}

	log.Printf("Connected to cluster! Server responded: %s", res)
	return client, nil
}

func (v *ValkeyCommand) GET(ctx context.Context, key string) any {
	val, err := v.Client.Get(ctx, key)
	if err != nil {
		fmt.Println("Error Fetching the value for the key:", key)
		fmt.Println("ERR:", err)
		return nil
	}
	return val
}

func (v *ValkeyCommand) Set(ctx context.Context, key string, value string) any {
	val, err := v.Client.Set(ctx, key, value)
	if err != nil {
		fmt.Println("Error Setting the value for the key:", key)
		fmt.Println("ERR:", err)
		return nil
	}
	return val
}

func (v *ValkeyCommand) SetEx(ctx context.Context, key string, value string, expiry int) any {
	transaction := pipeline.NewClusterBatch(true).Set(key, value).Expire(key, time.Second*time.Duration(expiry))
	options := pipeline.NewClusterBatchOptions().
		WithTimeout(2 * time.Second)
	val, err := v.Client.ExecWithOptions(ctx, *transaction, true, *options)
	if err != nil {
		fmt.Println("Error Setting the value for the key, with expiry:", key, expiry)
		fmt.Println("ERR:", err)
		return nil
	}
	return val
}

func (v *ValkeyCommand) SetNx(ctx context.Context, key string, value string) any {
	val, err := v.Client.MSetNX(ctx, map[string]string{
		key: value,
	})
	if err != nil {
		fmt.Println("Error Setting the value for the key, if key doesn't exist:", key)
		fmt.Println("ERR:", err)
		return nil
	}
	return val
}

func (v *ValkeyCommand) Flushall(ctx context.Context) any {
	val, err := v.Client.FlushAll(ctx)
	if err != nil {
		fmt.Println("Error Flushing all the keys")
		fmt.Println("ERR:", err)
		return nil
	}
	return val
}
