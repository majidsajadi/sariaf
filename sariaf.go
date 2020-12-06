package sariaf

import (
	"context"
	"net/http"
	"strings"
)

type node struct {
	path     string
	key      string
	children map[string]*node
	handler  http.HandlerFunc
	param    string
}

func (n *node) add(path string, handler http.HandlerFunc) {
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		var param string
		if len(k) > 1 && string(k[0]) == ":" {
			param = strings.TrimPrefix(k, ":")
			k = "*"
		}

		next, ok := current.children[k]
		if !ok {
			next = &node{
				path:     path,
				key:      k,
				children: make(map[string]*node),
				param:    param,
			}
			current.children[k] = next
		}
		current = next
	}

	current.handler = handler
}

func (n *node) find(path string) (*node, paramsType) {
	params := make(paramsType)
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		var next *node

		next, ok := current.children[k]
		if !ok {
			next, ok = current.children["*"]
			if !ok {
				return nil, params
			}

		}

		current = next

		if current.param != "" {
			params[current.param] = k
		}
	}

	return current, params
}

type Router struct {
	trees       map[string]*node
	middlewares []func(http.HandlerFunc) http.HandlerFunc
}

func New() *Router {
	return &Router{
		trees: make(map[string]*node),
	}
}

func (r *Router) Handle(method string, path string, handler http.HandlerFunc) {
	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &node{
			path:     "/",
			children: make(map[string]*node),
		}
	}

	r.trees[method].add(path, handler)
}

type contextKeyType struct{}
type paramsType map[string]string

var contextKey = contextKeyType{}

func newContext(ctx context.Context, params paramsType) context.Context {
	return context.WithValue(ctx, contextKey, params)
}

func fromContext(ctx context.Context) (paramsType, bool) {
	values, ok := ctx.Value(contextKey).(paramsType)

	return values, ok
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if _, ok := r.trees[req.Method]; !ok {
		http.NotFound(w, req)
		return
	}

	node, params := r.trees[req.Method].find(req.URL.Path)

	if node != nil && node.handler != nil {
		if len(params) != 0 {
			ctx := newContext(req.Context(), params)
			req = req.WithContext(ctx)
		}

		var handler = node.handler
		for _, middleware := range r.middlewares {
			handler = middleware(handler)
		}

		handler(w, req)
		return
	}

	http.NotFound(w, req)
}

func Params(r *http.Request) (paramsType, bool) {
	return fromContext(r.Context())
}

func (r *Router) Use(middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	if len(middlewares) > 0 {
		r.middlewares = append(r.middlewares, middlewares...)
	}
}
