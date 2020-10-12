# healthchecker
A dead simple health checker for GO services. 

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/regiocom/healthchecker)
[![Go Report Card](https://goreportcard.com/badge/github.com/regiocom/healthchecker)](https://goreportcard.com/report/github.com/regiocom/healthchecker)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/regiocom/healthchecker)](https://pkg.go.dev/github.com/regiocom/healthchecker) 

## TL;DR

Checks the availability of all services your service depends on and provides `/.well-known/alive` and `/.well-known/ready` endpoints. Supports some probes out of the box and can be extended by your own readiness probe. See [all available probes](https://pkg.go.dev/github.com/regiocom/healthchecker#Probe) or create a [custom probe](#custom-probes). 

Learn more [about health checks](#about-heath-checks).

## Usage

**Serve on same port with your application**
```go
func main() {
	checker := health.Checker{}

	// This can be any http.ServerMux
    checker.AppendHealthEndpoints(http.DefaultServeMux)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Hello World!"))
	})

	_ = http.ListenAndServe(":8080", http.DefaultServeMux)
}
```

**Serve on separate port** 
```go
func main() {
    checker := &health.Checker{}
    defer checker.ServeHTTPBackground(":8080")()

    // Check an external gRPC Service
    cc, _ := grpc.Dial(...)
    checker.AddReadinessProbe("my-grpc-service", health.GrpcProbe(cc))
}
```

## Custom Probes

A `health.Probe` is just a plain function returning an `error` if the service can not be reached. The probe is called any time the readiness endpoint is called. Thus use the most simple way to check if the service you depend on is up and running.

```go
// Checks if the customServiceConn can be reached.
func MyCustomServiceProbe(srv *customService) health.Probe {
    return func() error {
        // Check your service for availability
        available, err := srv.Ping()
        if !available || err != nil {
	    // Service is not available. We return an error.
            return fmt.Errorf("service is unavailable: %v", err)
	}
	
	// My depending service is up and running!
	return nil
    }
}
```

**Usage**
```go
checker := &health.Checker{}

src := NewCustomService()
checker.AddReadinessProbe("my-service", MyCustomServiceProbe(srv))
```

## Integrate with Kubernetes

This package is designed to seamlessly integrate with kubernetes. Lets asume we have a container image named `company/my-service` which uses this package. Than you can add the following lines to your deployment to enable monitoring by kubernetes. Kubernetes is now automatically restarting your service if it does not return alive in three consequitive requests. Also the pod is skipped by the load balancer as long it isn't ready. Learn more [about health checks](#about-heath-checks).

```diff
apiVersion: apps/v1
kind: Deployment
metadata:
  name: service1
spec:
  selector:
    matchLabels:
      app: service1
  template:
    metadata:
      labels:
        app: service1
    spec:
      containers:
        - name: service1
          image: company/my-service:latest
          ports:
            - containerPort: 80
+          livenessProbe:
+            httpGet:
+              path: /.well-known/alive
+              port: 80
+          readinessProbe:
+            httpGet:
+              path: /.well-known/ready
+              port: 80
```

---

## About Heath Checks

Kubernetes distinguishes between [liveliness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-a-liveness-command) and [readiness](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/#define-readiness-probes) checks. Thus, our services should provide two endpoints, one to check if the service is **alive** and one to check if the service is **ready**. 

**Alive**

A service is defined as alive, if it started correctly and accepts incoming requests. A service which is not alive for more than three times in a row will be killed and automatically restarted. 

**Ready**

A service is defined as ready, if all mandatory dependent services, for example a databases, can be reached and the service can work as expected. A service which is not ready for more than three times in a row will be skipped by the internal load balancer.

A service which is alive, but not ready has to recover itself.

> ℹ Per default both states are checked **every 2 seconds** after an initial delay of **10 seconds**. If a service needs more than 5 seconds to come up (alive=true), you should increase the initial delay to twice the mean startup time.

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

A service must implement a health endpoint to check if it is alive and ready. Both have to be served via `HTTP/1.1` under the same port on **all** interfaces. The routes should be `/.well-known/alive` and `/.well-known/ready`. Those endpoints must not require any authentication or any additional header. Response should either be `200 OK` or `503 Service Unavailable` and a minimal JSON body.

Both endpoints should be served independently and next to the main application on a different port.

### Alive

The response for the liveliness probe should be a simple true or false.

**`/.well-known/alive`: success**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
	"alive": true
}
```

**`/.well-known/alive`: failure**

```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{
	"alive": false
}
```

### Ready

The response for the readiness probe should be a simple true or false. For debug purpose the failure response can contain a list of simple reasons, why a service is unhealthy. Detailed information should be reported via the metrics / telemetry endpoints.

**`/.well-known/ready`: success**

```http
HTTP/1.1 200 OK
Content-Type: application/json

{
	"ready": true
}
```

**`/.well-known/ready`: failure**

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

