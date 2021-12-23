package hz

import (
	"context"
	"github.com/hazelcast/hazelcast-go-client"
	"log"
)

func NewHzClient() *hazelcast.Client {
	config := hazelcast.Config{}
	return NewHzClientWithConfig(config)
}

func NewHzClientWithConfig(config hazelcast.Config) *hazelcast.Client {
	cc := &config.Cluster
	cc.Network.SetAddresses("172.17.0.2:5701")
	cc.Discovery.UsePublicIP = true
	ctx := context.TODO()

	client, err := hazelcast.StartNewClientWithConfig(ctx, config)
	if err != nil {
		log.Fatalf("Hazelcast client cannot be created! err: %v\n", err)
	}
	return client
}
