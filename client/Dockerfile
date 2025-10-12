# Etap 1: Budowanie aplikacji
# Zmieniamy wersję Go na nowszą, zgodną z go.mod
FROM golang:1.23 as builder

WORKDIR /app

# Kopiujemy pliki go.mod i go.sum i pobieramy zależności
COPY go.mod go.sum ./
RUN go mod download

# Kopiujemy resztę kodu źródłowego
COPY . .

# Budujemy aplikację
RUN CGO_ENABLED=0 GOOS=linux go build -o /client .


# Etap 2: Finalny, lekki obraz
FROM alpine:latest

WORKDIR /app

# Kopiujemy skompilowaną aplikację z etapu budowania
COPY --from=builder /client .

# Kopiujemy plik konfiguracyjny
COPY config.yaml .

EXPOSE 8081 8082

# Domyślna komenda, która uruchomi się, jeśli nie zostanie nadpisana w docker-compose
CMD ["/app/client"]