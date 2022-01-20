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

	"github.com/nevzatseferoglu/hz-go-service/hz"
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

	entries, err := storage.myMap.GetEntrySet(r.Context())
	if err != nil {
		http.Error(rw, "error on get map", http.StatusInternalServerError)
	}
	log.Println("My distributed map entries are: ")
	for _, entry := range entries {
		log.Printf("key: %v, value: %v", entry.Key, entry.Value)
	}

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
		ServiceName: "hz-go-service",
		Port:        8080,
		Timeout:     10 * time.Second,
	}
	return c
}

func newInMemoryHzStorage(ctx context.Context) (*InMemoryHzStorage, error) {
	newClient, err := hz.NewHzClient(ctx)
	if err != nil {
		return nil, err
	}
	newMap, err := newClient.GetMap(ctx, "myDistributedMap")
	if err != nil {
		return nil, err
	}
	return &InMemoryHzStorage{hzClient: newClient, myMap: newMap}, nil
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
	// default context
	ctx := context.TODO()
	var err error
	// set service config as default
	config = newDefaultConfig()
	// create new client instance
	storage, err = newInMemoryHzStorage(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// put current map
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
	// create server instance
	srv := newServer(router, config)
	// start http server
	go func() {
		log.Println("Starting server...")
		log.Fatal(srv.ListenAndServe())
	}()
	// handle potential incoming errors
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
