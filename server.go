package rest

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Server - http server struct.
type Server struct {
	Port int

	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	httpServer *http.Server
}

func handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("."))
}

// Run - will initialize server and run it on provided port.
func (s *Server) Run(router http.Handler) error {
	if s.Port == 0 {
		s.Port = 8080
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
		mux := http.NewServeMux()
		mux.HandleFunc("/ping", handler)
		mux.HandleFunc("/liveness", handler)
		router = mux
	}

	s.httpServer = &http.Server{
		Addr:              fmt.Sprintf(":%d", s.Port),
		Handler:           router,
		ReadHeaderTimeout: s.ReadHeaderTimeout,
		WriteTimeout:      s.WriteTimeout,
		IdleTimeout:       s.IdleTimeout,
	}

	log.Printf("[INFO] rest server started on %s", s.httpServer.Addr)

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// Shutdown - shutdown http server.
func (s *Server) Shutdown(ctx context.Context) error {
	log.Print("[INFO] shutdown rest server")

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if s.httpServer != nil {
		if err := s.httpServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}
