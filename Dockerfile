FROM golang:1.26-alpine AS build

WORKDIR /src

ENV CGO_ENABLED=0 GOFLAGS=-trimpath

ARG VERSION=dev

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build \
    -ldflags="-s -w -X 'codeberg.org/romanzipp/location-ingest/internal/buildinfo.Version=${VERSION}'" \
    -o /out/location-ingest ./cmd/location-ingest


FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/location-ingest /usr/local/bin/location-ingest

ENV CONFIG=/etc/location-ingest/config.yaml

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/usr/local/bin/location-ingest"]
CMD ["--config", "/etc/location-ingest/config.yaml"]
