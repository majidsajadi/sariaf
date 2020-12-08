# Sariaf

Fast, simple and lightweight HTTP router for golang

## Install

`go get -u github.com/majidsajadi/sariaf`

## Features

- **Lightweight**
- **compatible with net/http**
- **No external dependencies**
- **Panic handler**
- **Custom not found handler**
- **No external dependencies**
- **Middleware support**
- **URL parameters**

## Usage

```go
package main

import (
	"net/http"

	"github.com/majidsajadi/sariaf"
)

func main() {
    r := sariaf.New()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("welcome"))
    })

    http.ListenAndServe(":3000", r)
}
```

## Advanced Usage 

```go
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/majidsajadi/sariaf"
)

func main() {
	router := sariaf.New()

	router.SetNotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Not Found")
	})

	router.SetPanicHandler(func(w http.ResponseWriter, r *http.Request, err interface{}) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Println("error:", err)
		fmt.Fprint(w, "Internal Server Error")
	})

	router.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})

	router.GET("/posts", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET: Get All Posts"))
	})

	router.GET("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.GetParams(r)
		w.Write([]byte("GET: Get Post With ID:" + params["id"]))
	})

	router.POST("/posts", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POST: Create New Post"))
	})

	router.PATCH("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.GetParams(r)
		w.Write([]byte("PATCH: Update Post With ID:" + params["id"]))
	})

	router.PUT("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.GetParams(r)
		w.Write([]byte("PUT: Update Post With ID:" + params["id"]))
	})

	router.DELETE("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.GetParams(r)
		w.Write([]byte("DELETE: Delete Post With ID:" + params["id"]))
	})

	router.GET("/error", func(w http.ResponseWriter, r *http.Request) {
		panic("Some Error Message")
	})

	log.Fatal(http.ListenAndServe(":8181", router))
}
```