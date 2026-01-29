# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install git for go mod download
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build server binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server

# Build keyctl binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /keyctl ./cmd/keyctl

# Runtime stage
FROM gcr.io/distroless/static:nonroot

WORKDIR /

# Copy binaries from builder
COPY --from=builder /server /server
COPY --from=builder /keyctl /keyctl

USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/server"]
