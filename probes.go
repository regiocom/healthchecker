package health

import (
	"fmt"

	"github.com/gomodule/redigo/redis"
	"github.com/nats-io/go-nats"
	"google.golang.org/grpc/connectivity"
)

type GrpcStateReporter interface {
	GetState() connectivity.State
}

// Checks a grpc connection for readiness.
//
// Example:
//		cc, _ := grpc.Dial(...)
//		checker.AddHealthyProbe("my-grpc-service", health.GrpcProbe(cc))
func GrpcProbe(conn GrpcStateReporter) Probe {
	return func() error {
		state := conn.GetState()
		if state != connectivity.Ready {
			return fmt.Errorf("grpc connection is in unready state: %v", state)
		}

		return nil
	}
}

type NatsStateReporter interface {
	Status() nats.Status
}

// Checks a nats connection for readiness.
//
// Example:
//		sc, _ := stan.Connect(...)
//		checker.AddHealthyProbe("my-stan-service", health.NatsProbe(sc.NatsConn()))
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
