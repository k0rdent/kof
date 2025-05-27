package server

import (
	"context"
	"encoding/json"
	"fmt"
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

const BasicInternalErrorMessage = "Something went wrong"

func (res *Response) Send(content any, code int) {
	jsonResponse, err := json.Marshal(content)
	if err != nil {
		res.Logger.Error(err, "failed to marshal response")
		res.Fail(BasicInternalErrorMessage, http.StatusInternalServerError)
		return
	}

	res.SetContentType("application/json")
	res.SetStatus(code)

	if _, err = fmt.Fprintln(res.Writer, string(jsonResponse)); err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}

func (res *Response) Fail(content string, code int) {
	res.Status = code
	http.Error(res.Writer, content, code)
}

func (res *Response) SetStatus(code int) {
	res.Status = code
	res.Writer.WriteHeader(code)
}

func (res *Response) SetContentType(contentType string) {
	res.Writer.Header().Set("Content-Type", contentType)
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
