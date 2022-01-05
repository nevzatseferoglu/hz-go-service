package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hazelcast/hazelcast-go-client"

	"github.com/nevzatseferoglu/sample-application/hz"
)

var (
	config    *ServiceConfig
	storage   *InMemoryHzStorage
	endpoints = [...]string{"config", "map"}
)

type InMemoryHzStorage struct {
	hzClient *hazelcast.Client
	myMap    *hazelcast.Map
}

type ServiceConfig struct {
	ServiceName string        `json:"serviceName"`
	Port        int           `json:"port"`
	Timeout     time.Duration `json:"timeout"`
}

func getConfigHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	hzClient := storage.hzClient

	serviceConfigResponse := struct {
		ServiceConfig
		hz.ClientInfo
	}{*config, hz.ClientInfo{Name: hzClient.Name(), Running: hzClient.Running()}}

	rspJson, err := json.Marshal(serviceConfigResponse)
	if err != nil {
		log.Fatal(err)
	}

	_, err = rw.Write(rspJson)
	if err != nil {
		log.Fatal(err)
	}
}

func getMapHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	log.Println(storage.myMap)

	//rspJson, err := json.Marshal(storage.myMap)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//
	//_, err = rw.Write(rspJson)
	//if err != nil {
	//	log.Fatalf("Problem with writing!, %v\n", err)
	//}
}

func healthHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func readinessHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func newDefaultConfig() *ServiceConfig {
	// default config properties
	c := &ServiceConfig{
		ServiceName: "sample-application",
		Port:        8080,
		Timeout:     10 * time.Second,
	}

	return c
}

func newInmemoryHzStorage(ctx context.Context) (*InMemoryHzStorage, error) {
	storage.hzClient, _ = hz.NewHzClient(ctx)

	var err error
	storage.myMap, err = storage.hzClient.GetMap(ctx, "myDistributedMap")
	if err != nil {
		return nil, err
	}

	return &InMemoryHzStorage{hzClient: storage.hzClient, myMap: storage.myMap}, nil
}

func newServer(router *mux.Router, config *ServiceConfig) *http.Server {
	return &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", config.Port),
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	}
}

func main() {
	ctx := context.TODO()

	var err error

	// set config as default
	config = newDefaultConfig()

	storage, err = newInmemoryHzStorage(ctx)
	if err != nil {
		log.Fatal(err)
	}

	_, err = storage.myMap.Put(ctx, "John", 1)
	if err != nil {
		log.Fatal(err)
	}

	// creates a server
	router := mux.NewRouter()

	// health checks for kubernetes
	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/readiness", readinessHandler)

	// service configuration handler
	router.HandleFunc(fmt.Sprintf("/%s/get", endpoints[0]), getConfigHandler)

	// hazelcast map handler
	router.HandleFunc(fmt.Sprintf("/%s/get", endpoints[1]), getMapHandler)

	srv := newServer(router, config)

	go func() {
		log.Println("Starting server...")
		log.Fatal(srv.ListenAndServe())
	}()

	handleSignal(srv)
}

func handleSignal(srv *http.Server) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// block until receiving signals
	<-signalChan

	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server cannot be properly shut down! err: %v\n", err)
	}

	log.Println("Server has been shut down!")
	os.Exit(0)
}
