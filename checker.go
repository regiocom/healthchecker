package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Probe func() error

type readyResponse struct {
	Ready   bool     `json:"ready"`
	Reasons []string `json:"reasons,omitempty"`
}

// A Checker can be used to provide a liveliness and readiness endpoint for your application.
// Use `checker.AddReadinessProbe` to add a test for readiness.
type Checker struct {
	readinessProbes map[string]Probe
	server          *http.Server
}

// Add a probe which should be run each time the service is checked for readiness.
// Example:
//		conn, _ := grpc.Dial(...)
//		checker.AddReadinessProbe("eventstore", health.GrpcProbe(conn))
func (h *Checker) AddReadinessProbe(service string, probe Probe) {
	_, alreadyRegistered := h.readinessProbes[service]
	if alreadyRegistered {
		panic("a health probe should have a unique identifier")
	}

	if h.readinessProbes == nil {
		h.readinessProbes = map[string]Probe{}
	}

	h.readinessProbes[service] = probe
}

// Serves health status endpoints via http
func (h *Checker) ServeHTTP(addr string) error {
	if h.server != nil {
		return fmt.Errorf("server is alrady running at %v", h.server.Addr)
	}

	h.server = &http.Server{Addr: addr, Handler: h.serverMux()}
	if err := h.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("could not listen on %s: %v", addr, err)
	}

	return nil
}

// Serves health endpoint in background. Calls os.Exit(1) in error.
// Use with defer to graceful shutdown the server.
// Example:
//	func main() {
//		health := &Checker{}
//		defer health.ServeHTTPBackground(8080)()
// 	}
func (h *Checker) ServeHTTPBackground(addr string) func() {
	go func() {
		err := h.ServeHTTP(addr)
		if err != nil {
			log.Fatalf("failed to start health server: %v", err)
		}
	}()

	return func() {
		err := h.Shutdown()
		if err != nil {
			log.Fatalf("failed to shutdown health server: %v", err)
		}
	}
}

// Gracefully stops health checker
func (h *Checker) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	return h.server.Shutdown(ctx)
}

func (h *Checker) serverMux() *http.ServeMux {
	m := http.NewServeMux()

	m.HandleFunc("/alive", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"alive":true}`))
	})

	m.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		ok, reasons := runProbes(h.readinessProbes)

		resp := &readyResponse{
			Ready:   ok,
			Reasons: reasons,
		}

		w.Header().Set("Content-Type", "application/json")

		if !resp.Ready {
			w.WriteHeader(http.StatusServiceUnavailable)
		}

		if b, err := json.Marshal(resp); err == nil {
			_, _ = w.Write(b)
		} else {
			log.Printf("failed to write health-check response: %v\n", err)
		}
	})

	return m
}

// Runs through all probes in parallel and returns ok and a list of reasons
func runProbes(probes map[string]Probe) (bool, []string) {
	wg := sync.WaitGroup{}
	m := sync.Mutex{}
	var reasons []string

	for service, probe := range probes {
		wg.Add(1)

		probe := probe
		service := service
		go func() {
			if err := probe(); err != nil {
				m.Lock()
				reasons = append(reasons, fmt.Sprintf("%v: %v", service, err))
				m.Unlock()
			}

			wg.Done()
		}()
	}

	wg.Wait()

	return len(reasons) == 0, reasons
}
