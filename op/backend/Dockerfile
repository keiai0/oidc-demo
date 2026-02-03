# =============================================================================
# Development stage (with Air hot reload)
# =============================================================================
FROM golang:1.26-alpine AS dev

RUN apk add --no-cache git curl

RUN go install github.com/air-verse/air@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

EXPOSE 8080

CMD ["air", "-c", ".air.toml"]

# =============================================================================
# Build stage
# =============================================================================
FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

# =============================================================================
# Production stage
# =============================================================================
FROM gcr.io/distroless/static-debian12 AS production

COPY --from=builder /server /server

EXPOSE 8080

ENTRYPOINT ["/server"]
