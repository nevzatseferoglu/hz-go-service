package hz

import (
	"context"
	"fmt"

	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/logger"
)

const withoutKubernetes = true

type ClientInfo struct {
	ClientName    string `json:"clientName"`
	ClientRunning bool   `json:"clientRunning"`
}

func NewHzClient(ctx context.Context) (*hazelcast.Client, error) {
	config := hazelcast.Config{
		ClientName: "hz-go-service-client",
	}
	cc := &config.Cluster
	if true {
		cc.Network.SetAddresses(fmt.Sprintf("%s:%s", "localhost", "5701"))
	} else {
		cc.Network.SetAddresses(fmt.Sprintf("%s:%s", "hazelcast.default.svc", "5701"))
	}
	cc.Discovery.UsePublicIP = false
	cc.Unisocket = true
	config.Logger.Level = logger.InfoLevel
	client, err := NewHzClientWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func NewHzClientWithConfig(ctx context.Context, config hazelcast.Config) (*hazelcast.Client, error) {
	client, err := hazelcast.StartNewClientWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
