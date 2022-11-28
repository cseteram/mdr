FROM golang:1.19-alpine3.16 AS builder
WORKDIR /app
COPY src/go.mod .
COPY src/go.sum .
RUN go mod download
COPY src .
RUN go build -o mdr

FROM alpine:3.16
WORKDIR /app
COPY --from=builder /app/mdr .
CMD ["/app/mdr"]
