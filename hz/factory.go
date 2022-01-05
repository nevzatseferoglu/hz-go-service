package hz

import (
	"context"
	"log"
	"time"

	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/cluster"
	"github.com/hazelcast/hazelcast-go-client/logger"
	"github.com/hazelcast/hazelcast-go-client/types"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientInfo struct {
	Name    string `json:"name"`
	Running bool   `json:"running"`
}

func getHazelcastClusterIP() (string, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", err
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}

	hzService, err := clientset.CoreV1().Services("default").Get(context.TODO(), "hazelcast", metav1.GetOptions{})
	if errors.IsNotFound(err) {
		log.Printf("Service hazelcast not found in default namespace\n")
		return "", err
	} else if statusError, isStatus := err.(*errors.StatusError); isStatus {
		log.Printf("Error getting service %v\n", statusError.ErrStatus.Message)
		return "", err
	} else if err != nil {
		return "", err
	} else {
		log.Printf("Found hazelcast service in default namespace\n")
	}

	return hzService.Spec.ClusterIP, nil
}

func NewHzClient(ctx context.Context) (*hazelcast.Client, error) {
	config := hazelcast.Config{}
	return NewHzClientWithConfig(ctx, config)
}

func NewHzClientWithConfig(ctx context.Context, config hazelcast.Config) (*hazelcast.Client, error) {
	config.ClientName = "my-go-client"
	config.SetLabels()

	cc := &config.Cluster
	cc.Name = "dev"
	cc.HeartbeatTimeout = types.Duration(5 * time.Second)
	cc.HeartbeatInterval = types.Duration(60 * time.Second)
	cc.InvocationTimeout = types.Duration(120 * time.Second)
	cc.RedoOperation = false
	cc.Unisocket = false
	cc.SetLoadBalancer(cluster.NewRoundRobinLoadBalancer())

	// cluster address
	base, err := getHazelcastClusterIP()
	if err != nil {
		return nil, err
	}
	cc.Network.SetAddresses(base + ":5701")
	cc.Discovery.UsePublicIP = false

	config.Stats.Enabled = false
	config.Stats.Period = types.Duration(5 * time.Second)

	config.Logger.Level = logger.TraceLevel

	client, err := hazelcast.StartNewClientWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
