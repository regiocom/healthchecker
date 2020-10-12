package health

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestChecker_alive(t *testing.T) {
	checker := &Checker{}
	server := httptest.NewServer(checker.serverMux())
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%v/alive", server.URL))

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(body), "true")
}

func TestChecker_AddHealthyProbe(t *testing.T) {
	called := false

	checker := &Checker{}
	checker.AddReadinessProbe("my-service", func() error {
		called = true
		return nil
	})

	server := httptest.NewServer(checker.serverMux())
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%v/ready", server.URL))

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(body), "true")

	assert.True(t, called)
}

func TestChecker_AddHealthyProbe_unhealthy(t *testing.T) {
	checker := &Checker{}
	checker.AddReadinessProbe("my-service", func() error {
		return fmt.Errorf("unhealthy")
	})

	server := httptest.NewServer(checker.serverMux())
	defer server.Close()

	resp, err := http.Get(fmt.Sprintf("%v/ready", server.URL))

	assert.NoError(t, err)
	assert.EqualValues(t, http.StatusServiceUnavailable, resp.StatusCode)
	body, _ := ioutil.ReadAll(resp.Body)
	assert.Contains(t, string(body), "my-service: unhealthy")
}
