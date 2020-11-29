package sariaf

import (
	"net/http"
	"strings"
)

type node struct {
	path     string
	key      string
	children map[string]*node
	handler  http.HandlerFunc
}

func (n *node) add(path string, handler http.HandlerFunc) {
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		next, ok := current.children[k]
		if !ok {
			next = &node{
				path:     path,
				key:      k,
				children: make(map[string]*node),
			}
			current.children[k] = next
		}
		current = next
	}

	current.handler = handler
}

func (n *node) find(path string) *node {
	current := n

	trimmed := strings.TrimPrefix(path, "/")
	slice := strings.Split(trimmed, "/")

	for _, k := range slice {
		next, ok := current.children[k]
		if !ok {
			return nil
		}
		current = next
	}
	return current
}

type Router struct {
	trees map[string]*node
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

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if _, ok := r.trees[req.Method]; !ok {
		http.NotFound(w, req)
		return
	}

	node := r.trees[req.Method].find(req.URL.Path)

	if node.path == req.URL.Path && node.handler != nil {
		node.handler(w, req)
		return
	}

	http.NotFound(w, req)
}
