package rest

import (
	"github.com/go-chi/chi/v5/middleware"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"
)

// Logger - log all requests
func Logger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, 1)
		start := time.Now()

		defer func() {
			statusCode := ww.Status()
			if statusCode == 0 {
				statusCode = 200
			}

			uri := r.URL.String()
			if qun, e := url.QueryUnescape(uri); e == nil {
				uri = qun
			}

			duration := time.Now().Sub(start)

			log.Printf("[DEBUG] %s - %s - %s - %v - %v", r.Method, uri, GetAddr(r), statusCode, duration)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

// Readiness - middleware for the readiness probe
func Readiness(endpoint string, isReady *atomic.Value) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" && strings.EqualFold(r.URL.Path, endpoint) {
				if isReady == nil || !isReady.Load().(bool) {
					ErrorResponse(w, r, http.StatusServiceUnavailable, nil, "")
					return
				}

				OkResponse(w)
				return
			}

			h.ServeHTTP(w, r)
		})
	}
}
