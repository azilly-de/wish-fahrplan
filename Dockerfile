# Build stage
FROM golang:1.24-alpine AS builder

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache ca-certificates git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o wish-fahrplan .

# Runtime stage
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

RUN adduser -D -h /app appuser
WORKDIR /app

COPY --from=builder /app/wish-fahrplan .

RUN mkdir -p /app/.ssh && chown -R appuser:appuser /app
USER appuser

EXPOSE 23234

ENTRYPOINT ["./wish-fahrplan"]
