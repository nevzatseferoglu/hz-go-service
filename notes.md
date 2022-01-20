```yml
# liveness and readiness check

livenessProbe: # prevents deadlock
httpGet:
  path: /health
  port: 8080
  scheme: HTTP
initialDelaySeconds: 5
periodSeconds: 15
timeoutSeconds: 5
readinessProbe: # prevents deadlock
httpGet:
  path: /readiness
  port: 8080
  scheme: HTTP
initialDelaySeconds: 5
timeoutSeconds: 1
```
minikube service --url hz-go-servic