package rest

import (
	"fmt"
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
	t.Run("default value", func(t *testing.T) {
		t.Run("address *", func(t *testing.T) {
			srv := &Server{
				Address: "*",
				Port:    8123,
			}

			handler := new(handlerFunc)
			go func() {
				require.NoError(t, srv.Run(handler))
			}()

			req, err := http.Get(fmt.Sprintf("http://localhost:%d", srv.Port))
			require.NoError(t, err)
			require.NotNil(t, req)

			require.Equal(t, http.StatusOK, req.StatusCode)
		})
		t.Run("empty port", func(t *testing.T) {
			srv := &Server{}

			handler := new(handlerFunc)
			go func() {
				require.NoError(t, srv.Run(handler))
			}()

			req, err := http.Get(fmt.Sprintf("http://localhost:8080"))
			require.NoError(t, err)
			require.NotNil(t, req)

			require.Equal(t, http.StatusOK, req.StatusCode)
		})
		t.Run("empty router", func(t *testing.T) {
			srv := &Server{
				Port: 1234,
			}
			defer func() {
				require.NoError(t, srv.Shutdown())
			}()

			go func() {
				require.NoError(t, srv.Run(nil))
			}()

			host := fmt.Sprintf("http://localhost:%d", srv.Port)
			resp, err := http.Get(host + "/ping")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = http.Get(host + "/liveness")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			resp, err = http.Get(host + "/readiness")
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, resp.StatusCode)

			srv.IsReady.Store(false)

			resp, err = http.Get(host + "/readiness")
			require.NoError(t, err)
			require.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
		})
	})

	// go run $GOROOT/src/crypto/tls/generate_cert.go --rsa-bits 1024 --host 127.0.0.1,::1,localhost --ca --start-date "Jan 1 00:00:00 1970" --duration=1000000h
	t.Run("SSL", func(t *testing.T) {
		t.Run("no cert file *", func(t *testing.T) {
			srv := &Server{
				SSL: &SSLConfig{},
			}
			require.Error(t, srv.Run(nil))
		})
		t.Run("no key file *", func(t *testing.T) {
			srv := &Server{
				SSL: &SSLConfig{
					CertPath: "./cert.pem",
				},
			}
			require.Error(t, srv.Run(nil))
		})
		//t.Run("run on default port", func(t *testing.T) {
		//	srv := &Server{
		//		Port: 5489,
		//		SSL: &SSLConfig{
		//			CertPath: "./cert.pem",
		//			KeyPath:  "./key.pem",
		//		},
		//	}
		//	defer func() {
		//		require.NoError(t, srv.Shutdown())
		//	}()
		//	go func() {
		//		require.NoError(t, srv.Run(nil))
		//	}()
		//
		//	req, err := http.NewRequest("GET", "https://localhost:5490/ping", nil)
		//	require.NoError(t, err)
		//	require.NotNil(t, req)
		//
		//	client := &http.Client{
		//		Transport: &http.Transport{
		//			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//		},
		//	}
		//	resp, err := client.Do(req)
		//	require.NoError(t, err)
		//	require.NotNil(t, req)
		//
		//	require.Equal(t, http.StatusOK, resp.StatusCode)
		//})
		//t.Run("redirect http to https", func(t *testing.T) {
		//	srv := &Server{
		//		Port: 4816,
		//		SSL: &SSLConfig{
		//			Redirect: true,
		//			URL:      "https://localhost:4817",
		//			CertPath: "./cert.pem",
		//			KeyPath:  "./key.pem",
		//		},
		//	}
		//	defer func() {
		//		require.NoError(t, srv.Shutdown())
		//	}()
		//	go func() {
		//		require.NoError(t, srv.Run(nil))
		//	}()
		//
		//	req, err := http.NewRequest("GET", "http://localhost:4816/ping?param=1", nil)
		//	require.NoError(t, err)
		//	require.NotNil(t, req)
		//	client := &http.Client{
		//		CheckRedirect: func(req *http.Request, via []*http.Request) error {
		//			return http.ErrUseLastResponse
		//		},
		//		Transport: &http.Transport{
		//			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//		},
		//	}
		//	resp, err := client.Do(req)
		//	require.NoError(t, err)
		//	require.NotNil(t, req)
		//	require.Equal(t, http.StatusTemporaryRedirect, resp.StatusCode)
		//})
	})
}

func TestServer_Shutdown(t *testing.T) {
	srv := &Server{
		Port: 9092,
	}
	handler := new(handlerFunc)

	go func() {
		time.Sleep(10 * time.Millisecond)
		require.NoError(t, srv.Shutdown())
	}()

	require.NoError(t, srv.Run(handler))
}
