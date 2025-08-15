# MongoDB + PostgreSQL Monitoring Stack

Kompletny stack monitoringu MongoDB i PostgreSQL z Prometheus i Grafana używający najnowszych wersji.

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
- **PostgreSQL**: localhost:5432 (admin/password123/testdb)
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

Po uruchomieniu dostępne będą dashboardy w Grafana:

**MongoDB:**
1. MongoDB Dashboard (ID: 2583)
2. MongoDB Overview (ID: 14997)
3. MongoDB Exporter (ID: 17016)

**PostgreSQL:**
1. PostgreSQL Dashboard (ID: 9628)
2. PostgreSQL Overview (ID: 455)

## 💾 Przykładowe dane

### MongoDB
MongoDB zostanie zainicjalizowany z przykładowymi danymi w bazie `testdb`:
- Kolekcja `users` - przykładowi użytkownicy
- Kolekcja `products` - przykładowe produkty

### PostgreSQL
PostgreSQL zostanie zainicjalizowany z przykładowymi danymi w bazie `testdb`:
- Tabela `users` - przykładowi użytkownicy
- Tabela `products` - przykładowe produkty
- View `products_summary` - podsumowanie produktów

## 🧪 Testowanie

### MongoDB
```bash
# Połączenie z MongoDB
docker exec -it mongodb mongosh -u admin -p password123 --authenticationDatabase admin

# W mongo shell:
use testdb
db.users.find()
db.products.find()
```

### PostgreSQL
```bash
# Połączenie z PostgreSQL
docker exec -it postgresql psql -U admin -d testdb

# W psql:
SELECT * FROM users;
SELECT * FROM products;
SELECT * FROM products_summary;
```

## 🔧 Szybkie dodanie danych

### MongoDB - dodanie nowego użytkownika:
```bash
docker exec -it mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval 'db = db.getSiblingDB("testdb"); db.users.insertOne({name: "Nowy Użytkownik", email: "nowy@example.com", age: 30, city: "Lublin"}); print("Dodano nowego użytkownika do MongoDB")'
```

### PostgreSQL - dodanie nowego użytkownika:
```bash
docker exec -it postgresql psql -U admin -d testdb -c "INSERT INTO users (name, email, age, city) VALUES ('Nowy Użytkownik', 'nowy@example.com', 30, 'Lublin'); SELECT 'Dodano nowego użytkownika do PostgreSQL' AS status;"
```

## 🔧 Diagnostyka

### Sprawdzenie statusu kontenerów:
```bash
docker-compose ps
```

### Sprawdzenie logów:
```bash
# PostgreSQL Exporter
docker-compose logs postgresql-exporter

# PostgreSQL
docker-compose logs postgresql

# Prometheus
docker-compose logs prometheus
```

### Sprawdzenie metryk:
```bash
# Sprawdź czy PostgreSQL Exporter zwraca metryki
curl http://localhost:9187/metrics

# Sprawdź targets w Prometheus
curl -s http://localhost:9090/api/v1/targets | jq '.data.activeTargets[] | {job, instance, health}'

# Sprawdź czy PostgreSQL jest dostępny
docker exec postgresql pg_isready -U admin -d testdb
```

### Restart PostgreSQL Exporter (jeśli potrzebny):
```bash
docker-compose restart postgresql-exporter
sleep 10
curl http://localhost:9187/metrics
```

## 📊 Sprawdzenie metryk

```bash
# Sprawdź metryki MongoDB
curl http://localhost:9216/metrics

# Sprawdź metryki PostgreSQL
curl http://localhost:9187/metrics

# Sprawdź targets w Prometheus
curl http://localhost:9090/api/v1/targets
```

## ⚠️ Uwagi

- Używane są najnowsze wersje (latest tags)
- Domyślne hasła są ustawione dla celów developmentu
- W produkcji zmień hasła i konfigurację bezpieczeństwa
- Porty: MongoDB (27017), PostgreSQL (5432), Prometheus (9090), Grafana (3000)
- Exportery: MongoDB (9216), PostgreSQL (9187)