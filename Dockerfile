FROM golang:1.25.3 as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download && go mod verify

COPY *.go ./

RUN CGO_ENABLED=0 GOOS=linux go build -o client -a -ldflags="-s -w" -installsuffix cgo

FROM scratch

WORKDIR /app

COPY --from=builder /app/client .
COPY config.yaml .

EXPOSE 8081 8082 8083

CMD ["/app/client"]