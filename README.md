# Sariaf

Fast, simple and lightweight HTTP router for golang

## Install

`go get -u github.com/bingoohuang/sariaf`

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
func main() {
	r := sariaf.New()

	r.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	http.ListenAndServe(":3000", r)
}
```

## Advanced Usage

```go
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
		params, _ := sariaf.Params(r)
		w.Write([]byte("GET: Get Post With ID:" + params["id"]))
	})

	// will match /start, /start/hello, /start/hello/world
	router.GET("/start/*action", func(w http.ResponseWriter, r *http.Request) {
		action := sariaf.Param(r, "action")
		w.Write([]byte("GET: action:" + action))
	})

	router.POST("/posts", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POST: Create New Post"))
	})

	router.PATCH("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.Params(r)
		w.Write([]byte("PATCH: Update Post With ID:" + params["id"]))
	})

	router.PUT("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.Params(r)
		w.Write([]byte("PUT: Update Post With ID:" + params["id"]))
	})

	router.DELETE("/posts/:id", func(w http.ResponseWriter, r *http.Request) {
		params, _ := sariaf.Params(r)
		w.Write([]byte("DELETE: Delete Post With ID:" + params["id"]))
	})

	router.GET("/error", func(w http.ResponseWriter, r *http.Request) {
		panic("Some Error Message")
	})

	log.Fatal(http.ListenAndServe(":8181", router))
}
```
