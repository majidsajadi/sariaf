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
    r := sariaf.NewRouter()

    r.Get("/", func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("welcome"))
    })

    http.ListenAndServe(":3000", r)
}
```

