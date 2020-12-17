package sariaf_test

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bingoohuang/sariaf"
	"github.com/stretchr/testify/assert"
)

// nolint:errcheck
func TestExample(t *testing.T) {
	r := sariaf.New()
	r.SetNotFound(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "Not Found", 404) })
	r.SetPanicHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
		http.Error(w, fmt.Sprintf("Internal Server Error:%v", err), 500)
	})

	r.GET("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Hello World")) })
	r.GET("/posts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("GET: Get All Posts")) })
	r.GET("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET: Get Post With ID:" + sariaf.Param(r, "id")))
	})
	// will match /start, /start/hello, /start/hello/world
	r.GET("/start/hello/*action", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET: action:" + sariaf.Param(r, "action")))
	})
	r.POST("/posts", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("POST: Create New Post")) })
	r.PATCH("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PATCH: Update Post With ID:" + sariaf.Param(r, "id")))
	})
	r.PUT("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUT: Update Post With ID:" + sariaf.Param(r, "id")))
	})
	r.DELETE("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE: Delete Post With ID:" + sariaf.Param(r, "id")))
	})
	r.GET("/error", func(w http.ResponseWriter, r *http.Request) { panic("Some Error Message") })

	r.GET("/abc/efg", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("GET: /abc/efg")) })

	l, _ := net.Listen("tcp", ":0")
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: r}
	defer server.Close()

	go server.ListenAndServe()

	base := fmt.Sprintf("http://127.0.0.1:%d", port)

	assertGet(t, http.MethodGet, base, 200, "Hello World")
	assertGet(t, http.MethodGet, base+"/posts", 200, "GET: Get All Posts")
	assertGet(t, http.MethodPost, base+"/posts", 200, "POST: Create New Post")
	assertGet(t, http.MethodPatch, base+"/posts/456", 200, "PATCH: Update Post With ID:456")
	assertGet(t, http.MethodPut, base+"/posts/456", 200, "PUT: Update Post With ID:456")
	assertGet(t, http.MethodDelete, base+"/posts/456", 200, "DELETE: Delete Post With ID:456")
	assertGet(t, http.MethodGet, base+"/posts/123", 200, "GET: Get Post With ID:123")
	assertGet(t, http.MethodGet, base+"/start/hello", 200, "GET: action:/")
	assertGet(t, http.MethodGet, base+"/start/hello/aaa", 200, "GET: action:/aaa")
	assertGet(t, http.MethodGet, base+"/start/hello/aaa/bbb", 200, "GET: action:/aaa/bbb")
	assertGet(t, http.MethodGet, base+"/start/hello/aaa/bbb", 200, "GET: action:/aaa/bbb")
	assertGet(t, http.MethodGet, base+"/error", 500, "Internal Server Error:Some Error Message\n")
	assertGet(t, http.MethodGet, base+"/notFound", 404, "Not Found\n")

	assertGet(t, http.MethodGet, base+"/abc/efg", 200, "GET: /abc/efg")
	assertGet(t, http.MethodGet, base+"/abc", 404, "Not Found\n")
}

func Rest(method, url string) (resp *http.Response, err error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(req)
}

func assertGet(t *testing.T, method, addr string, expectedState int, expectedBody string) {
	resp, err := Rest(method, addr)
	assert.Nil(t, err)

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	assert.Equal(t, expectedState, resp.StatusCode)
	assert.Equal(t, expectedBody, string(body))
}

func TestRouter(t *testing.T) {
	r := sariaf.New()
	assert.Nil(t, r.Handle(sariaf.MethodAny, "/any", nil))
	assertSearch(t, r, http.MethodGet, "/any", true, sariaf.RouterParams{}, nil)

	assert.Nil(t, r.Handle(http.MethodGet, "/", nil, sariaf.Tag("a")))
	assert.True(t, errors.Is(r.Handle(http.MethodGet, "/", nil), sariaf.ErrRouterDuplicate))
	assert.Nil(t, r.Handle(http.MethodGet, "/posts2", nil))
	assert.True(t, errors.Is(r.Handle(http.MethodGet, "/posts2/*id", nil, sariaf.Tag("idstar")), sariaf.ErrRouterSyntax))

	assert.Nil(t, r.Handle(http.MethodGet, "/posts/*id", nil, sariaf.Tag("idstar")))
	assert.True(t, errors.Is(r.Handle(http.MethodGet, "/posts/*id/*xx", nil), sariaf.ErrRouterSyntax))

	assert.True(t, errors.Is(r.Handle(http.MethodGet, "/posts/*id", nil), sariaf.ErrRouterDuplicate))
	assert.Nil(t, r.Handle(http.MethodPost, "/posts", nil))
	assert.Nil(t, r.Handle(http.MethodPatch, "/posts/:id", nil))
	assert.Nil(t, r.Handle(http.MethodPut, "/posts/:id", nil))
	assert.True(t, errors.Is(r.Handle(http.MethodPut, "/posts/:id", nil), sariaf.ErrRouterDuplicate))
	assert.Nil(t, r.Handle(http.MethodPut, "/posts/:id/:name", nil))
	assert.True(t, errors.Is(r.Handle(http.MethodPut, "/posts/:id:name", nil), sariaf.ErrRouterDuplicate))
	assert.Nil(t, r.Handle(http.MethodDelete, "/posts/:id", nil))
	assert.Nil(t, r.Handle(http.MethodGet, "/error", nil))

	assertSearch(t, r, http.MethodGet, "/", true, sariaf.RouterParams{}, "a")
	assertSearch(t, r, http.MethodGet, "/posts", true, sariaf.RouterParams{"id": "/"}, "idstar")
	assertSearch(t, r, http.MethodGet, "/posts/123", true, sariaf.RouterParams{"id": "/123"}, "idstar")
	assertSearch(t, r, http.MethodGet, "/posts/123/456", true, sariaf.RouterParams{"id": "/123/456"}, "idstar")
	assertSearch(t, r, http.MethodGet, "/others", false, sariaf.RouterParams{}, nil)
	assertSearch(t, r, http.MethodGet, "/any", true, sariaf.RouterParams{}, nil)
	assertSearch(t, r, http.MethodPost, "/any", true, sariaf.RouterParams{}, nil)
}

func assertSearch(t *testing.T, r *sariaf.Router, method, routerPath string,
	expectedFound bool, expectedParam sariaf.RouterParams, expectedTag interface{}) {
	node, params := r.Search(method, routerPath)
	assert.Equal(t, expectedFound, node != nil)
	assert.Equal(t, expectedParam, params)
	var nodeTag interface{}
	if node != nil {
		nodeTag = node.Option.Tag
	}
	assert.Equal(t, expectedTag, nodeTag)
}

func TestDuplicate(t *testing.T) {
	r := sariaf.New()

	assert.Nil(t, r.GET("/", http.NotFound))
	assert.True(t, errors.Is(r.GET("/", http.NotFound), sariaf.ErrRouterDuplicate))

	assert.Nil(t, r.GET("/:id", http.NotFound))
	assert.True(t, errors.Is(r.GET("/:id", http.NotFound), sariaf.ErrRouterDuplicate))
}

func TestInvalidSyntax(t *testing.T) {
	r := sariaf.New()

	assert.True(t, errors.Is(r.GET("/*a/*b", http.NotFound), sariaf.ErrRouterSyntax))
}

func TestStarName(t *testing.T) {
	r := sariaf.New()
	action := ""
	assert.Nil(t, r.GET("/*action", func(w http.ResponseWriter, r *http.Request) {
		action = sariaf.Param(r, "action")
	}))

	req, _ := http.NewRequest(http.MethodGet, "/hello", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "/hello", action)

	action = ""
	req, _ = http.NewRequest(http.MethodGet, "/hello/world", nil)
	r.ServeHTTP(httptest.NewRecorder(), req)

	assert.Equal(t, "/hello/world", action)
}
