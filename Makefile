.PHONY: default test install
all: default test install


tool:
	go get github.com/securego/gosec/cmd/gosec

sec:
	@gosec ./...
	@echo "[OK] Go security check was completed!"

init:
	export GOPROXY=https://goproxy.cn

lint:
	#golangci-lint run --enable-all
	golangci-lint run ./...

fmt:
	gofumports -w .
	gofumpt -w .
	gofmt -s -w .
	go mod tidy
	go fmt ./...
	revive .
	goimports -w .

install: init
	go install -ldflags="-s -w" ./...

linux: init
	GOOS=linux GOARCH=amd64 go install -ldflags="-s -w" ./...

test: init
	#go test -v ./...
	go test -v -race ./...

bench: init
	#go test -bench . ./...
	go test -tags bench -benchmem -bench . ./...

clean:
	rm coverage.out

cover:
	go test -v -race -coverpkg=./... -coverprofile=coverage.out ./...

coverview:
	go tool cover -html=coverage.out

