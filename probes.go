package health

import (
	"database/sql"
	"fmt"

	"github.com/gomodule/redigo/redis"
	vault "github.com/hashicorp/vault/api"
	"github.com/nats-io/go-nats"
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
