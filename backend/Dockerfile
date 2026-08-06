FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN apk add --no-cache git
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go test ./... && CGO_ENABLED=0 GOOS=linux go build -ldflags='-s -w' -o /bin/server ./cmd/api

FROM alpine:3.18
# create non-root user
RUN addgroup -S app && adduser -S -G app app
COPY --from=builder /bin/server /bin/server
COPY --from=builder /src/internal/storage/migrations /app/migrations
WORKDIR /app
RUN chown -R app:app /app /bin/server
USER app
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s CMD wget -qO- --tries=1 http://localhost:8080/health || exit 1
ENTRYPOINT ["/bin/server"]
