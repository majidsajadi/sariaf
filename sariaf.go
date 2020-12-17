package sariaf

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

// MethodAny means any http method.
const MethodAny = "ANY"

// ContextKeyType is the context key type.
type ContextKeyType int

const (
	// ContextKey is the context key for the params.
	ContextKey ContextKeyType = iota
)

type (
	// PanicHandlerType is the prototype for handler of panic.
	PanicHandlerType func(w http.ResponseWriter, req *http.Request, err interface{})
	// RouterParams is the type for request params.
	RouterParams map[string]string

	//  RouterOption is the option type for router.
	RouterOption struct {
		Tag interface{}
	}

	// RouterOptionFn defines the option func type to set router option.
	RouterOptionFn func(*RouterOption)

	// Node represents a sub path in the router trie.
	Node struct {
		Path     string
		Key      string
		Children map[string]*Node
		Handler  http.HandlerFunc
		Param    string
		Star     bool
		Option   *RouterOption
		Router   *Router
	}

	RouterContext struct {
		Params RouterParams
		Option *RouterOption
	}
)

var (
	// ErrRouterDuplicate is the root error for duplicate router path.
	ErrRouterDuplicate = errors.New("duplicate router path found")
	// ErrRouterSyntax is the root error for router pattern invalid syntax.
	ErrRouterSyntax = errors.New("invalid router syntax found")
)

// Tag attaches a tag to router option.
func Tag(tag interface{}) RouterOptionFn {
	return func(o *RouterOption) { o.Tag = tag }
}

// add method adds a new path to the trie.
func (n *Node) add(path string, handler http.HandlerFunc, r *Router, option *RouterOption) error {
	current := n
	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")
	duplicate := true
	stars := 0
	starPrev := false

	for _, k := range slice {
		if len(k) > 1 && k[0] == '*' {
			stars++
		}
		if stars > 1 {
			return fmt.Errorf("router pattern invalid, only one *abc allowed: %w", ErrRouterSyntax)
		}
	}

	for _, k := range slice {
		// replace keys with pattern ":abc" to "abc" or "*abc" to "abc" for matching params.
		param := ""
		if len(k) > 1 && (k[0] == ':' || k[0] == '*') {
			param = k[1:]
			k = "*"
		}

		if stars > 0 && starPrev {
			if _, ok := current.Children[k]; ok {
				break
			}
			return fmt.Errorf("router pattern %s conflicts: %w", path, ErrRouterSyntax)
		}

		next, ok := current.Children[k]
		if ok {
			starPrev = true
		} else {
			duplicate = false
			next = &Node{
				Path:     path,
				Key:      k,
				Children: make(map[string]*Node),
				Param:    param,
				Star:     stars > 0,
				Option:   option,
				Router:   r,
			}
			current.Children[k] = next
		}
		current = next
	}

	if duplicate {
		return fmt.Errorf("%s: %w", path, ErrRouterDuplicate)
	}

	current.Handler = handler
	return nil
}

// find method match the request url path with a Node in trie.
func (n *Node) find(path string) (*Node, RouterParams) {
	params := make(RouterParams)
	cur := n
	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for i, k := range slice {
		var next *Node

		next, ok := cur.Children[k]
		if !ok {
			if next, ok = cur.Children["*"]; !ok {
				// return nil if no Node match the given path.
				return nil, params
			}
		}

		cur = next

		// if the Node has a param add it to params map.
		if cur.Param != "" {
			if cur.Star {
				params[cur.Param] = filepath.Join(append([]string{"/"}, slice[i:]...)...)
				return cur, params
			}

			params[cur.Param] = k
		}
	}

	// fix pattern /*abc matching
	if next, ok := cur.Children["*"]; ok && next.Star {
		params[next.Param] = "/"
		cur = next
	}

	// avoid only parent matches without leaf.
	if cur.Handler == nil {
		cur = nil
	}

	// return the found Node and params map.
	return cur, params
}

// newContext returns a new Context that carries a provided params value.
func newContext(ctx context.Context, routerContext *RouterContext) context.Context {
	return context.WithValue(ctx, ContextKey, routerContext)
}

// fromContext extracts params from a Context.
func fromContext(ctx context.Context) *RouterContext {
	return ctx.Value(ContextKey).(*RouterContext)
}

// Router is an HTTP request multiplexer. It matches the URL of each
// incoming request against a list of registered path with their associated
// methods and calls the handler for the given URL.
type Router struct {
	trees map[string]*Node
	// middlewares stack.
	middlewares []func(http.HandlerFunc) http.HandlerFunc
	// notFound for when no matching route is found.
	notFound http.HandlerFunc
	// PanicHandler for handling panic.
	panicHandler PanicHandlerType
}

// NotFound replies to the request with an HTTP 404 not found error.
func NotFound(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusNotFound)
	_, _ = fmt.Fprint(w, "Sorry Not Found")
}

// PanicHandler replies to the request with an HTTP 500 and error message.
func PanicHandler(w http.ResponseWriter, r *http.Request, err interface{}) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = fmt.Fprint(w, "Internal Server Error:", err)
}

// New returns a new Router.
func New() *Router {
	return &Router{
		trees:        make(map[string]*Node),
		notFound:     NotFound,
		panicHandler: PanicHandler,
	}
}

// Noop replies to the request with nothing.
func Noop(http.ResponseWriter, *http.Request) {}

// ServeHTTP matches r.URL.Path with a stored route and calls handler for found Node.
func (n *Node) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			n.Router.panicHandler(w, req, err)
		}
	}()

	// call the middlewares on handler
	h := n.Handler
	for _, middle := range n.Router.middlewares {
		h = middle(h)
	}

	// call the Node handler
	h(w, req)
}

// ServeHTTP matches r.URL.Path with a stored route and calls handler for found Node.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// check if there is a trie for the request method.
	node, params := r.Search(req.Method, req.URL.Path)
	if node == nil {
		r.notFound(w, req)
		return
	}

	ctx := newContext(req.Context(), &RouterContext{
		Params: params,
		Option: node.Option,
	})

	node.ServeHTTP(w, req.WithContext(ctx))
}

// Search searches the node for specified http method and http request url path.
func (r *Router) Search(method, path string) (*Node, RouterParams) {
	// check if there is a trie for the request method.
	t, ok := r.trees[method]
	if !ok && method != MethodAny {
		t, ok = r.trees[MethodAny]
	}

	if !ok {
		return nil, nil
	}

	node, params := t.find(path)
	if node != nil || method == MethodAny {
		return node, params
	}

	// try any
	if t, ok = r.trees[MethodAny]; !ok {
		return nil, nil
	}

	return t.find(path)
}

// Handle registers a new path with the given path and method.
func (r *Router) Handle(method string, path string, handler http.HandlerFunc, optionFns ...RouterOptionFn) error {
	if handler == nil {
		handler = Noop
	}

	// check if for given method there is not any tie create a new one.
	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &Node{
			Path:     "/",
			Children: make(map[string]*Node),
			Router:   r,
		}
	}

	routerOption := &RouterOption{}
	for _, f := range optionFns {
		f(routerOption)
	}

	return r.trees[method].add(path, handler, r, routerOption)
}

// Params returns params stored in the request.
func Params(r *http.Request) RouterParams {
	return fromContext(r.Context()).Params
}

// Param returns param with name stored in the request.
func Param(r *http.Request, name string) string {
	params := Params(r)

	return params[name]
}

// Option returns router option with name stored in the request.
func Option(r *http.Request) *RouterOption {
	return fromContext(r.Context()).Option
}

// Use append middlewares to the middleware stack.
func (r *Router) Use(middlewares ...func(http.HandlerFunc) http.HandlerFunc) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// GET will register a path with a handler for get requests.
func (r *Router) GET(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodGet, path, handle, optionFns...)
}

// POST will register a path with a handler for post requests.
func (r *Router) POST(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodPost, path, handle, optionFns...)
}

// DELETE will register a path with a handler for delete requests.
func (r *Router) DELETE(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodDelete, path, handle, optionFns...)
}

// PUT will register a path with a handler for put requests.
func (r *Router) PUT(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodPut, path, handle, optionFns...)
}

// PATCH will register a path with a handler for patch requests.
func (r *Router) PATCH(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodPatch, path, handle, optionFns...)
}

// HEAD will register a path with a handler for head requests.
func (r *Router) HEAD(path string, handle http.HandlerFunc, optionFns ...RouterOptionFn) error {
	return r.Handle(http.MethodHead, path, handle, optionFns...)
}

// SetNotFound will register a handler for when no matching route is found
func (r *Router) SetNotFound(handle http.HandlerFunc) {
	r.notFound = handle
}

// SetPanicHandler will register a handler for handling panics
func (r *Router) SetPanicHandler(handle PanicHandlerType) {
	r.panicHandler = handle
}
