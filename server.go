package rest

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/pkg/errors"
	"log"
	"net/http"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

type SSLConfig struct {
	Port     int    // https server port
	Redirect bool   // defines if http requests will be redirected to the https
	URL      string // url where http requests will be redirected

	CertPath string // path to the ssl certificate
	KeyPath  string // path to the ssl key
}

// Server - rest server struct
type Server struct {
	Address string
	Port    int
	IsReady *atomic.Value
	SSL     *SSLConfig

	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	httpServer  *http.Server
	httpsServer *http.Server

	mu sync.Mutex
}

// Run - will initialize server and run it on provided port
func (s *Server) Run(router http.Handler) error {
	if s.Address == "*" {
		s.Address = ""
	}
	if s.Port == 0 {
		s.Port = 8080
	}

	if s.IsReady == nil {
		s.IsReady = &atomic.Value{}
		s.IsReady.Store(true)
	}

	if s.ReadHeaderTimeout == 0 {
		s.ReadHeaderTimeout = 10 * time.Second
	}
	if s.WriteTimeout == 0 {
		s.WriteTimeout = 30 * time.Second
	}
	if s.IdleTimeout == 0 {
		s.IdleTimeout = 60 * time.Second
	}

	if router == nil {
		mux := chi.NewRouter()
		mux.Use(Readiness("/readiness", s.IsReady))
		mux.HandleFunc("/ping", okHandler)
		mux.HandleFunc("/liveness", okHandler)
		router = mux
	}

	log.Printf("[INFO] http rest server on %s:%d", s.Address, s.Port)

	httpRouter := router

	if s.SSL != nil {
		if s.SSL.Port == 0 {
			s.SSL.Port = s.Port + 1
		}

		if s.SSL.Redirect {
			mux := chi.NewRouter()
			mux.Handle("/*", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				newURL := s.SSL.URL + r.URL.Path
				if r.URL.RawQuery != "" {
					newURL += "?" + r.URL.RawQuery
				}
				http.Redirect(w, r, newURL, http.StatusTemporaryRedirect)
			}))
			httpRouter = mux

			log.Printf("[INFO] http redirect server on %s:%d", s.Address, s.Port)
		}

		if _, err := os.Stat(s.SSL.CertPath); os.IsNotExist(err) {
			return errors.Wrap(err, "ssl certificate file not found")
		}
		if _, err := os.Stat(s.SSL.KeyPath); os.IsNotExist(err) {
			return errors.Wrap(err, "ssl key file not found")
		}

		log.Printf("[INFO] https rest server on %s:%d", s.Address, s.SSL.Port)

		s.mu.Lock()
		s.httpsServer = s.https(s.Address, s.SSL.Port, router)
		s.mu.Unlock()

		go func() {
			log.Printf("[WARN] https server terminated, %s", s.httpsServer.ListenAndServeTLS(s.SSL.CertPath, s.SSL.KeyPath))
		}()
	}

	s.mu.Lock()
	s.httpServer = s.http(s.Address, s.Port, httpRouter)
	s.mu.Unlock()

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return errors.Wrap(err, "start http server")
	}

	return nil
}

// Shutdown - shutdown rest server
func (s *Server) Shutdown() error {
	log.Print("[INFO] shutdown rest server")

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}
	if s.httpsServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()

		if err := s.httpsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) http(address string, port int, router http.Handler) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf("%s:%d", address, port),
		Handler:           router,
		ReadHeaderTimeout: s.ReadHeaderTimeout,
		WriteTimeout:      s.WriteTimeout,
		IdleTimeout:       s.IdleTimeout,
	}
}
func (s *Server) https(address string, port int, router http.Handler) *http.Server {
	server := s.http(address, port, router)
	server.TLSConfig = &tls.Config{
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		},
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
			tls.CurveP384,
		},
	}
	return server
}

func okHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("."))
}
