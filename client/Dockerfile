FROM golang:1.23 as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /client .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /client .

COPY config.yaml .

EXPOSE 8081 8082 8083

CMD ["/app/client"]