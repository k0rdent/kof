package server

import (
	"fmt"
	"net/http"
	"strings"
)

type Handler func(w *Response, r *http.Request)

type Router struct {
	routes          map[string]map[string]Handler
	server          *Server
	notFoundHandler Handler
}

func NewRouter(server *Server) *Router {
	return &Router{
		routes:          make(map[string]map[string]Handler),
		notFoundHandler: defaultNotFoundHandler,
		server:          server,
	}
}

func (r *Router) NotFound(handler Handler) {
	r.notFoundHandler = handler
}

func (r *Router) addRoute(method, path string, handler Handler) {
	if r.routes[method] == nil {
		r.routes[method] = make(map[string]Handler)
	}
	r.routes[method][path] = handler
}

func (r *Router) GET(path string, handler Handler) {
	r.addRoute(http.MethodGet, path, handler)
}

func (r *Router) POST(path string, handler Handler) {
	r.addRoute(http.MethodPost, path, handler)
}

func (r *Router) PUT(path string, handler Handler) {
	r.addRoute(http.MethodPut, path, handler)
}

func (r *Router) DELETE(path string, handler Handler) {
	r.addRoute(http.MethodDelete, path, handler)
}

func (r *Router) ServeHTTP(res *Response, req *http.Request) {
	handler, err := r.findHandler(req.Method, req.URL.Path)
	if err != nil {
		handler = r.notFoundHandler
	}
	if r.server != nil {
		handler = r.server.ApplyMiddleware(handler)
	}
	handler(res, req)
}

func (r *Router) findHandler(method, path string) (Handler, error) {
	if methodRoutes, ok := r.routes[method]; ok {
		if handler, ok := methodRoutes[path]; ok {
			return handler, nil
		}

		for routePath, handler := range methodRoutes {
			if isWildcardMatch(routePath, path) {
				return handler, nil
			}
		}
	}
	return nil, fmt.Errorf("no handler found for %s %s", method, path)
}

func isWildcardMatch(routePath, requestPath string) bool {
	routeParts := strings.Split(routePath, "/")
	requestParts := strings.Split(requestPath, "/")

	if len(routeParts) != len(requestParts) {
		return false
	}

	for i, part := range routeParts {
		if part == "*" {
			continue
		}
		if part != requestParts[i] {
			return false
		}
	}

	return true
}

func defaultNotFoundHandler(res *Response, req *http.Request) {
	http.Error(res.Writer, "404 page not found", http.StatusNotFound)
}
