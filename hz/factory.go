package hz

import (
	"context"
	"log"

	"github.com/hazelcast/hazelcast-go-client"
)

type ClientInfo struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
}

func NewHzClient() *hazelcast.Client {
	config := hazelcast.Config{}
	return NewHzClientWithConfig(config)
}

func NewHzClientWithConfig(config hazelcast.Config) *hazelcast.Client {
	cc := &config.Cluster
	cc.Network.SetAddresses("localhost:5701")
	cc.Discovery.UsePublicIP = true
	ctx := context.TODO()

	client, err := hazelcast.StartNewClientWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Hazelcast client cannot be created! err: %v\n", err)
	}
	return client
}
