BIN          := bin/location-ingest
ECHO_BIN     := bin/echo-upstream
PKG          := codeberg.org/romanzipp/location-ingest
VERSION      ?= dev
LDFLAGS      := -s -w -X '$(PKG)/internal/buildinfo.Version=$(VERSION)'
CONFIG       ?= config.example.yaml
HELM_CHART   := deploy/helm/location-ingest

.PHONY: run echo build test vet tidy fmt docker docker-run helm-lint helm-template clean

run:
	go run ./cmd/location-ingest --config $(CONFIG)

echo:
	go run ./cmd/echo-upstream

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o $(BIN) ./cmd/location-ingest

build-echo:
	CGO_ENABLED=0 go build -trimpath -o $(ECHO_BIN) ./cmd/echo-upstream

test:
	go test -count=1 ./...

vet:
	go vet ./...

tidy:
	go mod tidy

fmt:
	gofmt -s -w .

docker:
	docker build -t location-ingest:$(VERSION) --build-arg VERSION=$(VERSION) .

docker-run: docker
	docker run --rm -p 8080:8080 -v $(PWD)/$(CONFIG):/etc/location-ingest/config.yaml:ro location-ingest:$(VERSION)

helm-lint:
	helm lint $(HELM_CHART)

helm-template:
	helm template $(HELM_CHART)

clean:
	rm -rf bin dist
