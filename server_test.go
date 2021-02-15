package rest

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"testing"
	"time"
)

type handlerFunc struct{}

func (h *handlerFunc) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

func TestServer_Run(t *testing.T) {
	srv := &Server{
		Port: 8091,
	}
	ctx := context.Background()

	handler := new(handlerFunc)
	go func() {
		require.NoError(t, srv.Run(handler))
	}()

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d", srv.Port), nil)
	require.Nil(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode, "must be 200")

	go func() {
		time.Sleep(100 * time.Millisecond)
		require.NoError(t, srv.Shutdown(ctx))
	}()

	srv2 := &Server{
		Port: 8091,
	}
	err = srv2.Run(nil)
	require.Error(t, err)
}

func TestServer_Run_EmptyRouter(t *testing.T) {
	srv := &Server{
		Port: 1234,
	}
	defer func() {
		require.NoError(t, srv.Shutdown(context.Background()))
	}()

	go func() {
		require.NoError(t, srv.Run(nil))
	}()

	host := fmt.Sprintf("http://localhost:%d", srv.Port)
	resp, err := http.Get(host+"/ping")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	resp, err = http.Get(host+"/liveness")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestServer_Shutdown(t *testing.T) {
	srv := &Server{
		Port: 9092,
	}
	ctx := context.Background()
	handler := new(handlerFunc)

	go func() {
		time.Sleep(100 * time.Millisecond)
		require.NoError(t, srv.Shutdown(ctx))
	}()

	require.NoError(t, srv.Run(handler))
}
