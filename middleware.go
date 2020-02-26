package rest

import (
	"github.com/go-chi/chi/middleware"
	"log"
	"net/http"
	"net/url"
	"strings"
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

			log.Printf("[DEBUG] %s - %s - %s - %v - %v", r.Method, uri, ip, duration, statusCode)
		}()

		next.ServeHTTP(ww, r)
	}
	return http.HandlerFunc(fn)
}
