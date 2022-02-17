## hz-go-service
hazelcast go-client interaction on running hazelcast cluster in local cloud through minikube

## step by step
- `minikube start`
- `./operator.sh`
- Make sure that hazelcast members are running on the cloud by checking
  - `kubectl logs pod/hazelcast-0`
- The application can be deployed to that cluster.
  - `kubectl apply -f deployment.yml`
  
## Network
- Application tries to get an `ClusterIP` of running hazelcast service through kubernetes api. Then, expose itself to outside from the port `8081`.
