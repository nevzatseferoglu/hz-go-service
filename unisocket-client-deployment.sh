#!/bin/bash

export HAZELCAST_VERSION=5.0

kubectl apply -f https://raw.githubusercontent.com/hazelcast/hazelcast-kubernetes/master/rbac.yaml

kubectl run hz-hazelcast-0 --image=hazelcast/hazelcast:$HAZELCAST_VERSION -l "role=hazelcast"
kubectl run hz-hazelcast-1 --image=hazelcast/hazelcast:$HAZELCAST_VERSION -l "role=hazelcast"
kubectl run hz-hazelcast-2 --image=hazelcast/hazelcast:$HAZELCAST_VERSION -l "role=hazelcast"

kubectl create service loadbalancer hz-hazelcast --tcp=5701 -o yaml --dry-run=client | kubectl set selector --local -f - "role=hazelcast" -o yaml | kubectl create -f -

minikube tunnel
