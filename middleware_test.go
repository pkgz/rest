package rest

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/require"
	"log"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLogger(t *testing.T) {
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
	}()

	t.Run("basic", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/test-path")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		logOutput := buf.String()
		require.Contains(t, logOutput, "[DEBUG] GET")
		require.Contains(t, logOutput, "/test-path")
		require.Contains(t, logOutput, "200")
	})
	t.Run("encoded URL path", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/test%20path%20with%20spaces?jwt=secret-token&param=value")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		logOutput := buf.String()
		require.Contains(t, logOutput, "GET")
		require.Contains(t, logOutput, "/test path with spaces?")
		require.Contains(t, logOutput, "jwt=***")
		require.Contains(t, logOutput, "param=value")
	})
	t.Run("error status code", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})))
		defer ts.Close()

		resp, err := http.Get(ts.URL)
		require.NoError(t, err)
		require.Equal(t, http.StatusNotFound, resp.StatusCode)

		logOutput := buf.String()
		require.Contains(t, logOutput, "404")
	})

	t.Run("JWT token", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		defer ts.Close()

		jwtToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDM2OTA2OTEsImlhdCI6MTc0MzY4ODg5MH0.signature"
		resp, err := http.Get(ts.URL + "/test?jwt=" + jwtToken)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		logOutput := buf.String()
		fmt.Println(logOutput)
		require.Contains(t, logOutput, "[DEBUG] GET")
		require.Contains(t, logOutput, "/test?jwt=***")
		require.NotContains(t, logOutput, jwtToken)
	})
	t.Run("multiple query parameters with JWT", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		ts := httptest.NewServer(Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})))
		defer ts.Close()

		jwtToken := "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDM2OTA2OTF9.signature"
		resp, err := http.Get(ts.URL + "/api/data?param1=value1&jwt=" + jwtToken + "&param2=value2")
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode)

		logOutput := buf.String()
		require.Contains(t, logOutput, "/api/data?")
		require.Contains(t, logOutput, "param1=value1")
		require.Contains(t, logOutput, "jwt=***")
		require.Contains(t, logOutput, "param2=value2")
		require.NotContains(t, logOutput, jwtToken)
	})

	t.Run("malformed URL", func(t *testing.T) {
		var buf bytes.Buffer
		log.SetOutput(&buf)

		req := httptest.NewRequest("GET", "/valid", nil)
		req.URL = nil

		rec := httptest.NewRecorder()
		Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(rec, req)

		require.NotEmpty(t, buf.String())
	})

	t.Run("status code", func(t *testing.T) {
		testCases := []struct {
			name       string
			statusCode int
		}{
			{"OK", http.StatusOK},
			{"NotFound", http.StatusNotFound},
			{"BadRequest", http.StatusBadRequest},
			{"InternalServerError", http.StatusInternalServerError},
			{"ServiceUnavailable", http.StatusServiceUnavailable},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				log.SetOutput(&buf)

				handler := Logger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.statusCode)
				}))

				req := httptest.NewRequest("GET", "/status-code-test", nil)
				resp := httptest.NewRecorder()
				handler.ServeHTTP(resp, req)

				require.Equal(t, tc.statusCode, resp.Code)
				logOutput := buf.String()
				require.Contains(t, logOutput, fmt.Sprintf("%d", tc.statusCode))
			})
		}
	})
}

func TestReadiness(t *testing.T) {
	t.Run("service unavailable when not ready", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(false)

		handler := Readiness("/health", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	})

	t.Run("service available when ready", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(true)

		handler := Readiness("/health", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
	})

	t.Run("nil isReady value treated as not ready", func(t *testing.T) {
		handler := Readiness("/health", nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	})

	t.Run("passes through requests to other endpoints", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(false) // Even when not ready

		handler := Readiness("/health", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("GET", "/other-endpoint", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, "handler called", resp.Body.String())
	})

	t.Run("non-GET requests pass through at health endpoint", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(false)

		handler := Readiness("/health", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("POST", "/health", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusOK, resp.Code)
		require.Equal(t, "handler called", resp.Body.String())
	})

	t.Run("case insensitive path matching", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(false)

		handler := Readiness("/HeAlTh", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("handler called"))
		}))

		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()

		handler.ServeHTTP(resp, req)

		require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	})

	t.Run("toggle ready state", func(t *testing.T) {
		isReady := &atomic.Value{}
		isReady.Store(false)

		handler := Readiness("/health", isReady)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		// Not ready - should return 503
		req := httptest.NewRequest("GET", "/health", nil)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusServiceUnavailable, resp.Code)

		// Switch to ready - should return 200
		isReady.Store(true)
		req = httptest.NewRequest("GET", "/health", nil)
		resp = httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusOK, resp.Code)

		// Switch back to not ready - should return 503
		isReady.Store(false)
		req = httptest.NewRequest("GET", "/health", nil)
		resp = httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		require.Equal(t, http.StatusServiceUnavailable, resp.Code)
	})
}
