#!/bin/bash

export HAZELCAST_VERSION=5.0

kubectl apply -f https://raw.githubusercontent.com/hazelcast/hazelcast-kubernetes/master/rbac.yaml

kubectl create service loadbalancer hz-hazelcast-0 --tcp=5701
kubectl run hz-hazelcast-0 --image=hazelcast/hazelcast:$HAZELCAST_VERSION --port=5701 -l "app=hz-hazelcast-0,role=hazelcast"
kubectl create service loadbalancer hz-hazelcast-1 --tcp=5701
kubectl run hz-hazelcast-1 --image=hazelcast/hazelcast:$HAZELCAST_VERSION --port=5701 -l "app=hz-hazelcast-1,role=hazelcast"
kubectl create service loadbalancer hz-hazelcast-2 --tcp=5701
kubectl run hz-hazelcast-2 --image=hazelcast/hazelcast:$HAZELCAST_VERSION --port=5701 -l "app=hz-hazelcast-2,role=hazelcast"

kubectl create service loadbalancer hz-hazelcast --tcp=5701 -o yaml --dry-run=client | kubectl set selector --local -f - "role=hazelcast" -o yaml | kubectl create -f -

minikube tunnel
