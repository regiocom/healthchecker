package health

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gomodule/redigo/redis"
	vault "github.com/hashicorp/vault/api"
	"github.com/nats-io/go-nats"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"google.golang.org/grpc/connectivity"
)

// Interface matching a gRPC client's state method.
type GrpcStateReporter interface {
	GetState() connectivity.State
}

// Checks a grpc connection for readiness.
//
// Example:
//		cc, _ := grpc.Dial(...)
//		checker.AddReadinessProbe("my-grpc-service", health.GrpcProbe(cc))
func GrpcProbe(conn GrpcStateReporter) Probe {
	return func() error {
		state := conn.GetState()
		if state != connectivity.Ready {
			return fmt.Errorf("grpc connection is in unready state: %v", state)
		}

		return nil
	}
}

// Pings a http endpoint for readiness. Called endpoint should return 2xx as status.
// **INFO:** If you check another service using this lib, always use the `/.well-known/alive endpoint` to prevent cascading requests.
//
// Example:
//		checker.AddReadinessProbe("my-http-service", health.HTTPProbe("http://my-service:8080/.well-known/alive"))
func HTTPProbe(endpoint string) Probe {
	return func() error {
		// #nosec G107
		resp, err := http.Get(endpoint)
		if err != nil {
			return fmt.Errorf("endpoint could not be reached: %v", err)
		}

		if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
			return nil
		}

		return fmt.Errorf("service is not ready: %v - %v", resp.StatusCode, resp.Status)
	}
}

// Interface matching a mongodb client's ping method.
type MongoStateReporter interface {
	Ping(ctx context.Context, rp *readpref.ReadPref) error
}

// Checks a mongodb connection for readiness.
//
// Example:
//		client, _ := mongo.Connect(ctx, options.Client().ApplyURI(uri))
//		checker.AddReadinessProbe("my-mongo-client", health.MongoProbe(client))
func MongoProbe(client MongoStateReporter) Probe {
	return func() error {
		return client.Ping(context.Background(), readpref.Primary())
	}
}

// Interface matching a nats client's status method.
type NatsStateReporter interface {
	Status() nats.Status
}

// Checks a nats connection for readiness.
//
// Example:
//		sc, _ := stan.Connect(...)
//		checker.AddReadinessProbe("my-stan-service", health.NatsProbe(sc.NatsConn()))
func NatsProbe(conn NatsStateReporter) Probe {
	return func() error {
		state := conn.Status()

		if state != nats.CONNECTED {
			return fmt.Errorf("nats connection is in unready state: %v", state)
		}

		return nil
	}
}

// Checks a pool of redis connection for readiness.
func RedisPoolProbe(pool *redis.Pool) Probe {
	return func() error {
		err := pool.Get().Err()
		if err != nil {
			return fmt.Errorf("redis connection is not useable: %v", err.Error())
		}

		return nil
	}
}

// Checks a SQL connection for readiness.
func SQLProbe(db *sql.DB) Probe {
	return func() error {
		return db.Ping()
	}
}

// Interface matching a vault client's health method.
type VaultHealthReporter interface {
	Health() (*vault.HealthResponse, error)
}

// Checks a vault connection for readiness
func VaultProbe(hr VaultHealthReporter) Probe {
	return func() error {
		hc, err := hr.Health()
		if err != nil {
			return fmt.Errorf("could not get vault health: %v", err.Error())
		}

		if !hc.Initialized {
			return fmt.Errorf("vault is not initialized")
		}

		if hc.Sealed {
			return fmt.Errorf("vault is sealed")
		}

		if hc.Standby {
			return fmt.Errorf("vault is on standby")
		}

		return nil
	}
}
