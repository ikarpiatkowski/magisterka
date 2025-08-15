# MongoDB Monitoring Stack

Kompletny stack monitoringu MongoDB z Prometheus i Grafana używający najnowszych wersji.

## 🚀 Szybki start

```bash
# 1. Przejdź do katalogu
cd mongodb-monitoring

# 2. Nadaj uprawnienia
chmod +x setup.sh

# 3. Uruchom setup
./setup.sh
```

## 📊 Dostępne serwisy

- **MongoDB**: localhost:27017 (admin/password123)
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin123)

## 🛠️ Komendy Docker Compose

```bash
# Uruchomienie
docker-compose up -d

# Status
docker-compose ps

# Logi
docker-compose logs [service_name]

# Zatrzymanie
docker-compose down

# Zatrzymanie z usunięciem volumes
docker-compose down -v
```

## 📈 Dashboardy

Po uruchomieniu dostępne będą 3 dashboardy MongoDB w Grafana:
1. MongoDB Dashboard (ID: 2583)
2. MongoDB Overview (ID: 14997)  
3. MongoDB Exporter (ID: 17016)

## 💾 Przykładowe dane

MongoDB zostanie zainicjalizowany z przykładowymi danymi w bazie `testdb`:
- Kolekcja `users` - przykładowi użytkownicy
- Kolekcja `products` - przykładowe produkty

## 🧪 Testowanie

```bash
# Połączenie z MongoDB
docker exec -it mongodb mongosh -u admin -p password123 --authenticationDatabase admin

# W mongo shell:
use testdb
db.users.find()
db.products.find()
```

## ⚠️ Uwagi

- Używane są najnowsze wersje (latest tags)
- Domyślne hasła są ustawione dla celów developmentu
- W produkcji zmień hasła i konfigurację bezpieczeństwa