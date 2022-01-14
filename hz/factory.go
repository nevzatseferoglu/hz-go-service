package hz

import (
	"context"
	"log"

	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/logger"
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
	// getting config for the client inside kuberneetes cluster
	config, err := rest.InClusterConfig()
	if err != nil {
		return "", err
	}
	// creates clientset for given config
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", err
	}
	// return cluster services resources
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

	// cluster address
	// base, err := getHazelcastClusterIP()
	// if err != nil {
	//	return nil, err
	// }

	cc.Network.SetAddresses("hazelcast.default.svc" + ":5701")
	cc.Discovery.UsePublicIP = false

	config.Logger.Level = logger.TraceLevel

	client, err := hazelcast.StartNewClientWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}
	return client, nil
}
