package server

import (
	"context"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

type Server struct {
	addr         string
	Router       *Router
	middlewares  []Middleware
	errorHandler ErrorHandler
	server       *http.Server
	logger       *logr.Logger
}

type Response struct {
	Status   int
	Writer   http.ResponseWriter
	Duration time.Duration
	Logger   *logr.Logger
}

func (res *Response) Fail(content string, code int) {
	res.Status = code
	http.Error(res.Writer, content, code)
}

func (res *Response) SetStatus(code int) {
	res.Status = code
	res.Writer.WriteHeader(code)
}

type ServerOption func(*Server)

func NewServer(addr string, logger *logr.Logger) *Server {
	s := &Server{
		addr:        addr,
		middlewares: []Middleware{},
		logger:      logger,
	}
	s.Router = NewRouter(s)
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.Router.ServeHTTP(&Response{Writer: w, Logger: s.logger}, r)
}

func (s *Server) Run() error {
	s.server = &http.Server{
		Addr:    s.addr,
		Handler: s,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) GetRouter() *Router {
	return s.Router
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

func WithErrorHandler(handler ErrorHandler) ServerOption {
	return func(s *Server) {
		s.errorHandler = handler
	}
}

func DefaultErrorHandler(res *Response, r *http.Request, err error) {
	res.Fail(err.Error(), http.StatusInternalServerError)
}
