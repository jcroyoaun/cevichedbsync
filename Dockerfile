# Build the manager binary
FROM docker.io/golang:1.24 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
RUN go mod download

# Copy the go source
COPY cmd/main.go cmd/main.go
COPY api/ api/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# Install migrate CLI
RUN curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
RUN chmod +x migrate

# Final image
FROM alpine:latest
WORKDIR /

RUN apk add --no-cache postgresql-client

# Copy migrate CLI
COPY --from=builder /workspace/migrate /usr/local/bin/migrate

# Copy manager binary
COPY --from=builder /workspace/manager /manager

# Create a non-root user
RUN addgroup -g 65532 -S nonroot && adduser -u 65532 -S nonroot -G nonroot
USER 65532:65532

ENTRYPOINT ["/manager"]
