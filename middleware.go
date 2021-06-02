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

			ip := strings.Split(r.RemoteAddr, ":")[0]
			if strings.HasPrefix(r.RemoteAddr, "[") {
				ip = strings.Split(r.RemoteAddr, "]:")[0] + "]"
			}
			duration := time.Now().Sub(start)

			log.Printf("[DEBUG] %s - %s - %s - %v - %v", r.Method, uri, ip, statusCode, duration)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}

// Readiness - middleware for the readiness probe
func Readiness(isReady *atomic.Value) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if isReady == nil || !isReady.Load().(bool) {
			http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}
