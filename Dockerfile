# Build stage
FROM golang:1.22-alpine AS builder

# Install GCC and C-compiler tools needed for go-sqlite3
RUN apk add --no-cache build-base

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Enable CGO for SQLite
RUN CGO_ENABLED=1 GOOS=linux go build -o engine main.go

# Run stage
FROM alpine:latest
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/engine .

EXPOSE 8080
CMD ["./engine"]
