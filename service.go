package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/types"

	"github.com/nevzatseferoglu/hz-go-service/hz"
)

const hztimeout = time.Second * 3
const sampleMap = "myDistributedMap"

var (
	config    *ServiceConfig
	imdg      *InMemoryHzStorage
	hzc       *hazelcast.Client
	endpoints = [...]string{"config", "map"}
)

type Info struct {
	Msg string `json:"msg"`
}

type InMemoryHzStorage struct {
	myMap *hazelcast.Map
}

type ServiceConfig struct {
	ServiceName string        `json:"serviceName"`
	Port        int           `json:"port"`
	Timeout     time.Duration `json:"timeout"`
}

func getConfigHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	serviceConfigResponse := struct {
		ServiceConfig
		hz.ClientInfo
	}{*config, hz.ClientInfo{ClientName: hzc.Name(), ClientRunning: hzc.Running()}}
	rspJson, err := json.Marshal(serviceConfigResponse)
	if err != nil {
		log.Fatal(err)
	}
	_, err = rw.Write(rspJson)
	if err != nil {
		log.Fatal(err)
	}
}

func validateURLQueryParameter(r *http.Request) (mapNames, keys []string, err error) {
	// resolve query parameters
	if err := r.ParseForm(); err != nil {
		return nil, nil, err
	}
	var ok bool
	if mapNames, ok = r.Form["name"]; !ok {
		return nil, nil, errors.New("invalid request body")
	}
	if keys, ok = r.Form["key"]; !ok {
		return nil, nil, errors.New("invalid request body")
	}
	return mapNames, keys, nil
}

func mapEntryHandler(rw http.ResponseWriter, r *http.Request) {
	// set response type
	rw.Header().Set("Content-Type", "application/json")
	var err error
	mapNames, keys, err := validateURLQueryParameter(r)
	if err != nil {
		http.Error(rw, fmt.Sprintf("URL query parameters cannot be validated, err: %v.", err), http.StatusBadRequest)
		return
	}
	mapName := mapNames[0]
	key := keys[0]
	// wrap a timeout context for the hazelcast operation
	ctx, cancel := context.WithTimeout(r.Context(), hztimeout)
	defer cancel()
	objInfo, err := hzc.GetDistributedObjectsInfo(ctx)
	if err != nil {
		http.Error(rw, "Hazelcast objects info cannot be obtained", http.StatusInternalServerError)
		return
	}
	exist := false
	for _, o := range objInfo {
		if hazelcast.ServiceNameMap == o.ServiceName && mapName == o.Name {
			exist = true
			break
		}
	}
	switch r.Method {
	case "GET":
		rw.WriteHeader(http.StatusOK)
		// return proper response as a json value
		var rsp []byte
		if !exist {
			nonExistMap := Info{
				Msg: "There is no map for the given name",
			}
			rsp, err = json.Marshal(nonExistMap)
			if err != nil {
				http.Error(rw, fmt.Sprintf("marshal cannot work properly for non exist map response, rsp: %v\n", nonExistMap), http.StatusInternalServerError)
				return
			}
			_, err = rw.Write(rsp)
			return
		}
		// get map
		m, err := hzc.GetMap(ctx, mapName)
		if err != nil {
			http.Error(rw, fmt.Sprintf("map: %s cannot be returned from the hazelcast", mapName), http.StatusInternalServerError)
			return
		}
		// get value
		value, err := m.Get(ctx, key)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Value of key: %s cannot be returned from the hazelcast", key), http.StatusInternalServerError)
			return
		}
		if value == nil {
			nonExist := Info{
				Msg: "There is no value for the given key",
			}
			rsp, err = json.Marshal(nonExist)
			if err != nil {
				http.Error(rw, fmt.Sprintf("marshal cannot work properly for non exist value response, rsp: %v\n", nonExist), http.StatusInternalServerError)
				return
			}
		} else {
			entry := types.Entry{
				Key:   key,
				Value: value,
			}
			rsp, err = json.Marshal(entry)
			if err != nil {
				http.Error(rw, fmt.Sprintf("marshal cannot work properly for the existing value response, rsp: %v\n", entry), http.StatusInternalServerError)
				return
			}
		}
		_, err = rw.Write(rsp)
		if err != nil {
			http.Error(rw, fmt.Sprintf("response cannot be properly written\n"), http.StatusInternalServerError)
			return
		}
	case "POST":
		rw.WriteHeader(http.StatusCreated)
		if !isContentTypeJson(r) {
			http.Error(rw, "Invalid content type is given in header.\n", http.StatusBadRequest)
			return
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(rw, "response body cannot be read\n", http.StatusInternalServerError)
			return
		}
		type RequestBody struct {
			Value interface{} `json:"value"`
		}
		var reqBody RequestBody
		err = json.Unmarshal(body, &reqBody)
		if err != nil {
			http.Error(rw, "Invalid request body", http.StatusBadRequest)
			return
		}
		var rsp []byte
		pmap, err := hzc.GetMap(ctx, mapName)
		if err != nil {
			http.Error(rw, "Hazelcast cannot return a map\n", http.StatusInternalServerError)
			return
		}
		_, err = pmap.Put(ctx, key, reqBody.Value)
		if err != nil {
			http.Error(rw, fmt.Sprintf("Hazelcast cannot put %v value to map\n", reqBody.Value), http.StatusInternalServerError)
			return
		}
		successInfo := Info{
			Msg: fmt.Sprintf("(Key: %s, Value: %v) has been put to Map: %s successfully",
				key, reqBody.Value, mapName),
		}
		rsp, err = json.Marshal(successInfo)
		if err != nil {
			http.Error(rw, fmt.Sprintf("marshal cannot work properly for non exist map put, rsp: %v\n", successInfo), http.StatusInternalServerError)
			return
		}
		_, err = rw.Write(rsp)
	}
}

func isContentTypeJson(r *http.Request) bool {
	contentType := r.Header.Get("Content-type")
	if contentType == "" {
		return false
	}
	if contentType != "application/json" {
		return false
	}
	return true
}

func healthHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func readinessHandler(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func newDefaultServiceConfig() *ServiceConfig {
	// default config properties
	return &ServiceConfig{
		ServiceName: "hz-go-service",
		Port:        8080,
		Timeout:     5 * time.Second,
	}
}

func newInMemoryHzStorage(ctx context.Context, c *hazelcast.Client) (*InMemoryHzStorage, error) {
	entries := []types.Entry{
		{Key: "key1", Value: "value1"},
		{Key: "key2", Value: "value2"},
		{Key: "key3", Value: "value3"},
	}
	newMap, err := c.GetMap(ctx, sampleMap)
	err = newMap.PutAll(ctx, entries...)
	if err != nil {
		return nil, err
	}
	return &InMemoryHzStorage{myMap: newMap}, nil
}

func newServer(router *mux.Router, config *ServiceConfig) *http.Server {
	return &http.Server{
		Handler:      router,
		Addr:         fmt.Sprintf(":%d", config.Port),
		ReadTimeout:  config.Timeout,
		WriteTimeout: config.Timeout,
	}
}

func newRouter() *mux.Router {
	// creates a server
	router := mux.NewRouter()
	// health checks for kubernetes
	router.HandleFunc("/health", healthHandler)
	router.HandleFunc("/readiness", readinessHandler)
	// service configuration handler
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[0]), getConfigHandler)
	// hazelcast map handlers
	router.HandleFunc(fmt.Sprintf("/%s", endpoints[1]), mapEntryHandler)
	return router
}

func main() {
	ctx := context.Background()
	var err error
	// set service config as default
	config = newDefaultServiceConfig()
	// initiate a new client
	hzc, err = hz.NewHzClient(ctx)
	if err != nil {
		log.Fatal(err)
	}
	// create a new in-memory storage
	imdg, err = newInMemoryHzStorage(ctx, hzc)
	if err != nil {
		log.Fatal(err)
	}
	router := newRouter()
	// create server instance
	srv := newServer(router, config)
	// start http server
	go func() {
		log.Println("Server is listening...")
		log.Fatal(srv.ListenAndServe())
	}()
	// handle potential incoming errors
	handleSignal(ctx, srv)
}

func handleSignal(ctx context.Context, srv *http.Server) {
	// signal channel
	signalChan := make(chan os.Signal, 1)
	// add interrupts signals to signal channel
	signal.Notify(signalChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	// block until receiving signals
	<-signalChan
	// new context for limit the closing time
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	if hzc != nil {
		err := hzc.Shutdown(ctx)
		if err != nil {
			log.Fatalf("Client cannot be shut down after the signal is received, err: %v\n", err)
		}
	}
	err := srv.Shutdown(ctx)
	if err != nil {
		log.Fatalf("Server cannot be properly shut down after the signal is received, err: %v\n", err)
	}
	log.Println("Server has been shut down!")
	os.Exit(0)
}
