package health

import (
	"testing"

	"github.com/nats-io/go-nats"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/connectivity"
)

type MockGrpcReporter struct {
	state connectivity.State
}

func (m MockGrpcReporter) GetState() connectivity.State {
	return m.state
}

func TestGrpcProbe(t *testing.T) {
	reporter := &MockGrpcReporter{
		state: connectivity.Ready,
	}

	probe := GrpcProbe(reporter)

	assert.NoError(t, probe())
}

func TestGrpcProbe_err(t *testing.T) {
	reporter := &MockGrpcReporter{
		state: connectivity.Connecting,
	}

	probe := GrpcProbe(reporter)

	assert.Error(t, probe())
}

type MockNatsReporter struct {
	state nats.Status
}

func (m MockNatsReporter) Status() nats.Status {
	return m.state
}

func TestNatsProbe(t *testing.T) {
	reporter := &MockNatsReporter{
		state: nats.CONNECTED,
	}

	probe := NatsProbe(reporter)

	assert.NoError(t, probe())
}

func TestNatsProbe_err(t *testing.T) {
	reporter := &MockNatsReporter{
		state: nats.CONNECTING,
	}

	probe := NatsProbe(reporter)

	assert.Error(t, probe())
}
