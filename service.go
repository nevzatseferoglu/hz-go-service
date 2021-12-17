package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

const (
	appConfig      = "appConfig"
	appServiceName = "sample-application"
	appPort        = 8080
)

type State int

const (
	Unknown State = 1 << iota
	NotAvailable
	Available
)

var (
	appState State
)

func (s State) String() string {
	switch appState {
	case Unknown:
		return fmt.Sprintf("Unknown")
	case NotAvailable:
		return fmt.Sprintf("NotAvailable")
	case Available:
		return fmt.Sprintf("Available")
	}

	return ""
}

func appConfigHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	type appConfigResponse struct {
		ServiceName string `json:"serviceName"`
		StateInfo   string `json:"stateInfo"`
	}

	response := appConfigResponse{
		ServiceName: appServiceName,
		StateInfo:   appState.String(),
	}

	rspJson, err := json.Marshal(&response)
	if err != nil {
		log.Fatalf("Invalid JSON formatting!, %v\n", err)
		return
	}

	_, err = rw.Write(rspJson)
	if err != nil {
		log.Fatal("Problem with writing!\n")
		return
	}
}

func getEnvOrDefault(name string, def string) string {
	var ok bool
	var res string
	if res, ok = os.LookupEnv(name); !ok {
		res = def
	}
	return res
}

func setEnv(name string, value string) (string, error) {
	err := os.Setenv(name, value)
	if err != nil {
		log.Fatalf("%s:%s was note set properly!", name, value)
		return "", err
	}
	return os.Getenv(name), nil
}

func setState(curState *State, newState State) (State, error) {
	oldState := *curState
	if curState != nil {
		*curState = newState
		return oldState, nil
	}
	return -1, errors.New("currState cannot be empty")
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc(fmt.Sprintf("/%s", appConfig), appConfigHandler)

	if _, err := setState(&appState, Available); err != nil {
		log.Fatal(err)
		return
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", appPort), router))
}
