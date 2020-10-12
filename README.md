# healthchecker
A dead simple health checker for GO applications

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/regiocom/healthchecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/regiocom/healthchecker)](https://goreportcard.com/report/github.com/regiocom/healthchecker)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/regiocom/healthchecker)](https://pkg.go.dev/github.com/regiocom/healthchecker) 

## Usage
```go
func main() {
    health := &Checker{}
    defer health.ServeHTTPBackground(":8080")()

    // Check an external gRPC Service
    cc, _ := grpc.Dial(...)
    checker.AddReadinessProbe("my-grpc-service", health.GrpcProbe(cc))
}
```

## About Heath Checks

Kubernetes distinguishes between [liveliness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command) and [readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes) checks. Thus, our services should provide two endpoints, one to check if the service is **alive** and one to check if the service is **ready**. 

**Alive**

A service is defined as alive, if it started correctly and accepts incoming requests. A service which is not alive for more than three times in a row will be killed and automatically restarted. 

**Ready**

A service is defined as ready, if all mandatory dependent services, for example a databases, can be reached and the service can work as expected. A service which is not ready for more than three times in a row will be skipped by the internal load balancer.

A service which is alive, but not ready has to recover itself.

> Per default both states are checked **every 2 seconds** after an initial delay of **10 seconds**. If a service needs more than 5 seconds to come up (alive=true), you should increase the initial delay to twice the mean startup time.

### State decision matrix

The following table contains a set of common states / events and the expected health report.

| State                                     | Alive    | Ready    |
| ----------------------------------------- | -------- | -------- |
| Startup phase                             | false    | false    |
| Ready                                     | **true** | **true** |
| Deadlock                                  | false    | false    |
| Heavy load due to processing lots of data | **true** | false    |
| Database cannot be reached                | **true** | false    |
| Mandatory Service cannot be reached       | **true** | false    |
| Volume has insufficient space             | **true** | false    |
| Slow response from service / database     | **true** | **true** |
| Leader cannot be reached                  | **true** | false    |

## Implementation

A service must implement an health endpoint to check if it is alive and ready. Both have to be served via `HTTP/1.1` under the same port (default: 8080) on **all** interfaces. The routes should be `/alive` and `/ready`. Those endpoints must not require any authentication or any additional header. Response should either be `200 OK` or `503 Service Unavailable` and a minimal JSON body.

Both endpoints should be served independently and next to the main application on a different port.

### Alive

The response for the liveliness probe should be a simple true or false.

**`/alive`: success**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
	"alive": true
}
```

**`/alive`: failure**

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{
	"alive": false
}
```

### Ready

The response for the readiness probe should be a simple true or false. For debug purpose the failure response can contain a list of simple reasons, why a service is unhealthy. Detailed information should be reported via the metrics / telemetry endpoints.

**`/ready`: success**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
	"ready": true
}
```

**`/ready`: failure**

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{
	"ready": false,
	"reasons": [
		"dgraph: Service unreachable"
	]
}
```

