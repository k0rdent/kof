package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Middleware func(Handler) Handler

type CORSConfig struct {
	AllowOrigins     []string
	AllowMethods     []string
	AllowHeaders     []string
	AllowCredentials bool
	MaxAge           int
}

func Chain(h Handler, middleware ...Middleware) Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	return h
}

func (s *Server) ApplyMiddleware(h Handler) Handler {
	if len(s.middlewares) == 0 {
		return h
	}
	return Chain(h, s.middlewares...)
}

func (s *Server) Use(middleware ...Middleware) {
	s.middlewares = append(s.middlewares, middleware...)
}

func RecoveryMiddleware(next Handler) Handler {
	return func(res *Response, req *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				res.Logger.Error(err.(error), "Recovery")
				res.Fail("Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next(res, req)
	}
}

func LoggingMiddleware(next Handler) Handler {
	return func(res *Response, req *http.Request) {
		b := time.Now()
		next(res, req)
		res.Duration = time.Since(b)
		res.Logger.Info(fmt.Sprintf("\"%s %s\" %d %f", req.Method, req.URL, res.Status, res.Duration.Seconds()))
	}
}

func CORSMiddleware(config *CORSConfig) Middleware {
	if config == nil {
		config = DefaultCORSConfig()
	}

	allowOrigins := strings.Join(config.AllowOrigins, ", ")
	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	maxAge := ""
	if config.MaxAge > 0 {
		maxAge = fmt.Sprintf("%d", config.MaxAge)
	}

	return func(next Handler) Handler {
		return func(res *Response, req *http.Request) {
			res.Writer.Header().Set("Access-Control-Allow-Origin", allowOrigins)
			res.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
			res.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)

			if config.AllowCredentials {
				res.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if maxAge != "" {
				res.Writer.Header().Set("Access-Control-Max-Age", maxAge)
			}

			if req.Method == http.MethodOptions {
				res.SetStatus(http.StatusNoContent)
				return
			}

			next(res, req)
		}
	}
}

func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: true,
	}
}
