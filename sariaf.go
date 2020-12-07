// Copyright 2020 Majid Sajadi. All rights reserved.
// Use of this source code is governed by a MIT-style license that can be found
// in the LICENSE file.

package sariaf

import (
	"context"
	"net/http"
	"strings"
)

var methods = map[string]string{
	"GET":    http.MethodGet,
	"POST":   http.MethodPost,
	"PUT":    http.MethodPut,
	"DELETE": http.MethodDelete,
	"PATCH":  http.MethodPatch,
	"HEAD":   http.MethodHead,
}

// each node represent a path in the router trie.
type node struct {
	path     string
	key      string
	children map[string]*node
	handler  http.HandlerFunc
	param    string
}

// add method adds a new path to the trie.
func (n *node) add(path string, handler http.HandlerFunc) {
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		// replace keys with pattern ":*" with "*" for matching params.
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

// find method match the request url path with a node in trie.
func (n *node) find(path string) (*node, Params) {
	params := make(Params)
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		var next *node

		next, ok := current.children[k]
		if !ok {
			next, ok = current.children["*"]
			if !ok {
				// return nil if no node match the given path.
				return nil, params
			}

		}

		current = next

		// if the node has a param add it to params map.
		if current.param != "" {
			params[current.param] = k
		}
	}

	// return the found node and params map.
	return current, params
}

type panicHandlerType func(w http.ResponseWriter, req *http.Request, err interface{})

type contextKeyType struct{}

// Params is the type for request params.
type Params map[string]string

// contextKey is the context key for the params.
var contextKey = contextKeyType{}

// newContext returns a new Context that carries a provided params value.
func newContext(ctx context.Context, params Params) context.Context {
	return context.WithValue(ctx, contextKey, params)
}

// fromContext extracts params from a Context.
func fromContext(ctx context.Context) (Params, bool) {
	values, ok := ctx.Value(contextKey).(Params)

	return values, ok
}

// Router is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered path with their associated
// methods and calls the handler for the given URL.
type Router struct {
	trees map[string]*node
	// middlewares stack.
	middlewares []func(http.HandlerFunc) http.HandlerFunc
	// notFound for when no matching route is found.
	notFound http.HandlerFunc
	// PanicHandler for handling panic.
	panicHandler panicHandlerType
}

// New returns a new Router.
func New() *Router {
	return &Router{
		trees: make(map[string]*node),
	}
}

// ServeHTTP matches r.URL.Path with a stored route and calls handler for found node.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if r.panicHandler != nil {
		defer func() {
			if err := recover(); err != nil {
				r.panicHandler(w, req, err)
			}
		}()
	}

	// check if there is a trie for the request method.
	if _, ok := r.trees[req.Method]; !ok {
		http.NotFound(w, req)
		return
	}

	// find the node with request url path in the trie.
	node, params := r.trees[req.Method].find(req.URL.Path)

	if node != nil {
		// attach the params context to request if any param exists.
		if len(params) != 0 {
			ctx := newContext(req.Context(), params)
			req = req.WithContext(ctx)
		}

		// call the middlewares on handler
		var handler = node.handler
		for _, middleware := range r.middlewares {
			handler = middleware(handler)
		}

		// call the node handler
		handler(w, req)
		return
	}

	// call the not found handler if can match the request url path to any node in trie.
	if r.notFound != nil {
		r.notFound(w, req)
	} else {
		http.NotFound(w, req)
	}
}

// Handle registers a new path with the given path and method.
func (r *Router) Handle(method string, path string, handler http.HandlerFunc) {
	if _, ok := methods[method]; !ok {
		panic("method is not valid")
	}

	if handler == nil {
		panic("handle must not be nil")
	}

	// check if for given method there is not any tie create a new one.
	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &node{
			path:     "/",
			children: make(map[string]*node),
		}
	}

	r.trees[method].add(path, handler)
}

// GetParams returns params stored in the request.
func GetParams(r *http.Request) (Params, bool) {
	return fromContext(r.Context())
}

// Use append middlewares to the middleware stack.
func (r *Router) Use(middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	if len(middlewares) > 0 {
		r.middlewares = append(r.middlewares, middlewares...)
	}
}

// GET will register a path with a handler for get requests.
func (r *Router) GET(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodGet, path, handle)
}

// POST will register a path with a handler for post requests.
func (r *Router) POST(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodPost, path, handle)
}

// DELETE will register a path with a handler for delete requests.
func (r *Router) DELETE(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodDelete, path, handle)
}

// PUT will register a path with a handler for put requests.
func (r *Router) PUT(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodPut, path, handle)
}

// PATCH will register a path with a handler for patch requests.
func (r *Router) PATCH(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodPatch, path, handle)
}

// HEAD will register a path with a handler for head requests.
func (r *Router) HEAD(path string, handle http.HandlerFunc) {
	r.Handle(http.MethodHead, path, handle)
}

// SetNotFound will register a handler for when no matching route is found
func (r *Router) SetNotFound(handle http.HandlerFunc) {
	r.notFound = handle
}

// SetPanicHandler will register a handler for handling panics
func (r *Router) SetPanicHandler(handle panicHandlerType) {
	r.panicHandler = handle
}
