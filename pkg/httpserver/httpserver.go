package httpserver

import (
	"context"
	config "mdata/configs"
	"net/http"
	"time"
)

const (
	_defaultReadTimeout     = 5 * time.Second
	_defaultWriteTimeout    = 5 * time.Second
	_defaultAddr            = ":8080"
	_defaultShutdownTimeout = 3 * time.Second
)

// Server -.
type Server struct {
	server          *http.Server
	notify          chan error
	shutdownTimeout time.Duration
}

// New -.
func New(handler http.Handler, cfg config.HTTP) *Server {
	httpServer := &http.Server{
		Handler:      handler,
		ReadTimeout:  _defaultReadTimeout,
		WriteTimeout: _defaultWriteTimeout,
		Addr:         _defaultAddr,
		//ErrorLog:     log.ErrorLog, // вопрос: как лучше передавать сюда эту зависимость? или лучше здесь использовать стандартный логгер?
	}

	if time.Duration(cfg.HTTPReadTimeout) != _defaultReadTimeout {
		httpServer.ReadTimeout = time.Duration(cfg.HTTPReadTimeout * int(time.Second))
	}

	if time.Duration(cfg.HTTPWriteTimeout) != _defaultWriteTimeout {
		httpServer.WriteTimeout = time.Duration(cfg.HTTPWriteTimeout * int(time.Second))
	}

	if cfg.HTTPAddr != _defaultAddr {
		httpServer.Addr = cfg.HTTPAddr
	}

	s := &Server{
		server:          httpServer,
		notify:          make(chan error, 1),
		shutdownTimeout: _defaultShutdownTimeout,
	}

	s.start()

	return s
}

func (s *Server) start() {
	go func() {
		s.notify <- s.server.ListenAndServe()
		close(s.notify)
	}()
}

// Notify -.
func (s *Server) Notify() <-chan error {
	return s.notify
}

// Shutdown -.
func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()

	return s.server.Shutdown(ctx)
}

func (s *Server) GetAddr() string {
	return s.server.Addr
}
