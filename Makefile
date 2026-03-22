VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS = -s -w \
	-X github.com/SamuelJenkinsML/am-i-cooking/internal/version.Version=$(VERSION) \
	-X github.com/SamuelJenkinsML/am-i-cooking/internal/version.Commit=$(COMMIT) \
	-X github.com/SamuelJenkinsML/am-i-cooking/internal/version.Date=$(DATE)

.PHONY: build test clean install

build:
	go build -ldflags "$(LDFLAGS)" -o cook .

test:
	go test ./...

clean:
	rm -f cook

install:
	go install -ldflags "$(LDFLAGS)" .
