FROM golang:1.21-alpine AS builder
WORKDIR /src
COPY go.mod .
COPY . .
RUN apk add --no-cache git
RUN go build -o /bin/server ./

FROM alpine:3.18
COPY --from=builder /bin/server /bin/server
EXPOSE 8080
CMD ["/bin/server"]
