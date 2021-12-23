## sample-application
hazelcast go-client interaction with running hazelcast cluster in cloud

### Warning
```go
cc.Network.SetAddresses("<EXTERNAL-IP>:5701")
```
- `<EXTERNAL-IP>` is needed to be set explicitly as hardcoded. There is no auto-discovery mechanism to determine ip address. 

### smart-client-deployment
- https://guides.hazelcast.org/kubernetes-external-client/#_smart_client

### unisocket-client-deployment
- https://guides.hazelcast.org/kubernetes-external-client/#_unisocket_client

### ToDO
- Redirection on `config` and `map` endpoints.

### Dependencies
- Running hazelcast cluster somewhere else
