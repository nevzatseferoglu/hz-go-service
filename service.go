package main

import (
	"context"
	"encoding/json"
	"errors"
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
	endpoints = [...]string{"config", "setConfig", "getMap", "setMap"}
	queryMap  *hazelcast.Map
	hzClient  *hazelcast.Client
)

type ServiceConfig struct {
	ServiceName  string        `json:"serviceName"`
	State        State         `json:"state"`
	Port         int           `json:"port"`
	Timeout      int64         `json:"timeout"`
	HzClientInfo hz.ClientInfo `json:"hzClientInfo"`
}

type State int

const (
	Unknown State = 1 << iota
	NotAvailable
	Available
)

func (s State) String() string {
	switch s {
	case Unknown:
		return fmt.Sprintf("Unknown")
	case NotAvailable:
		return fmt.Sprintf("NotAvailable")
	case Available:
		return fmt.Sprintf("Available")
	}

	return ""
}

func setConfigHandler(rw http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if newServiceName := q.Get("serviceName"); newServiceName != "" {
		config.ServiceName = newServiceName
	}

	if newState := q.Get("state"); newState != "" {
		var s State
		switch newState {
		case "NotAvailable":
			s = NotAvailable
		case "Available":
			s = Available
		default:
			s = Unknown
		}

		_, _ = setState(&config.State, s)
	}
}

func getConfigHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

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

func healthHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func readinessHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func getEnvOrDefault(name string, def string) string {
	var ok bool
	var res string
	if res, ok = os.LookupEnv(name); !ok {
		res = def
	}
	return res
}

func setEnv(name string, value string) string {
	err := os.Setenv(name, value)
	if err != nil {
		log.Fatalf("%s:%s was note set properly!", name, value)
	}
	return os.Getenv(name)
}

func setState(curState *State, newState State) (State, error) {
	oldState := *curState
	if curState != nil {
		*curState = newState
		return oldState, nil
	}
	return -1, errors.New("currState cannot be empty")
}

func handleSignal(srv *http.Server) {
	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// block until receiving signals.
	<-interruptChan

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)
	defer cancel()
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server cannot be properly shut down! err: %v\n", err)
	}

	log.Println("Server has been shut down!")
	os.Exit(0)
}

func newDefaultConfig() *ServiceConfig {
	// default config properties
	c := &ServiceConfig{
		ServiceName: "sample-application",
		State:       Available,
		Port:        8080,
		Timeout:     10,
	}

	return c
}

func setupHzClientAndMap() {
	hzClient = hz.NewHzClient()

	var (
		ctx context.Context
		err error
	)
	if queryMap, err = hzClient.GetMap(ctx, "queryMap"); err != nil {
		log.Fatal(err)
	}
}

func setupService() {
	config = newDefaultConfig()
	setupHzClientAndMap()
}

func setMapHandler(rw http.ResponseWriter, r *http.Request) {
	var ctx context.Context

	values := r.URL.Query()
	for key, item := range values {
		if err := queryMap.Set(ctx, key, item); err != nil {
			log.Fatal(err)
		}
	}
}

func getMapHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	rspJson, err := json.Marshal(queryMap)
	if err != nil {
		log.Fatal(err)
	}

	_, err = rw.Write(rspJson)
	if err != nil {
		log.Fatalf("Problem with writing!, %v\n", err)
	}
}

func main() {
	// set default config
	setupService()

	// creates a server
	router := mux.NewRouter()
	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/readiness", readinessHandler)
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[0]), getConfigHandler)
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[1]), setConfigHandler)
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[3]), setMapHandler)
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[2]), getMapHandler)

	srv := &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", config.Port),
		ReadTimeout:  time.Duration(config.Timeout) * time.Second,
		WriteTimeout: time.Duration(config.Timeout) * time.Second,
	}

	go func() {
		log.Println("Starting server...")
		log.Fatal(srv.ListenAndServe())
	}()

	handleSignal(srv)
}
