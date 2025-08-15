#!/bin/bash

echo "🚀 Konfiguracja MongoDB + PostgreSQL Monitoring Stack"

# Tworzenie struktury katalogów
echo "📁 Tworzenie struktury katalogów..."
mkdir -p grafana/provisioning/datasources
mkdir -p grafana/provisioning/dashboards
mkdir -p grafana/dashboards

# Pobieranie gotowych dashboardów MongoDB z Grafana Labs
echo "📊 Pobieranie oficjalnego dashboardu MongoDB..."
curl -s https://grafana.com/api/dashboards/2583/revisions/2/download > grafana/dashboards/mongodb-dashboard.json

echo "📊 Pobieranie alternatywnego dashboardu MongoDB Overview..."
curl -s https://grafana.com/api/dashboards/14997/revisions/1/download > grafana/dashboards/mongodb-overview.json

echo "📊 Pobieranie dashboardu MongoDB Exporter..."
curl -s https://grafana.com/api/dashboards/17016/revisions/1/download > grafana/dashboards/mongodb-exporter.json

# Pobieranie dashboardów PostgreSQL
echo "📊 Pobieranie oficjalnego dashboardu PostgreSQL..."
curl -s https://grafana.com/api/dashboards/9628/revisions/7/download > grafana/dashboards/postgresql-dashboard.json

echo "📊 Pobieranie alternatywnego dashboardu PostgreSQL Overview..."
curl -s https://grafana.com/api/dashboards/455/revisions/2/download > grafana/dashboards/postgresql-overview.json

# Tworzenie sieci Docker (jeśli nie istnieje)
echo "🌐 Tworzenie sieci Docker..."
docker network create mongodb-monitoring_monitoring 2>/dev/null || echo "Sieć już istnieje"

# Uruchamianie stacka
echo "🐳 Uruchamianie Docker Compose..."
docker-compose up -d

echo "⏳ Czekanie na uruchomienie wszystkich serwisów..."
sleep 30

echo "🔍 Sprawdzanie statusu serwisów..."
echo "MongoDB:"
docker exec mongodb mongosh --eval "db.adminCommand('ismaster')" --quiet 2>/dev/null && echo "  ✅ MongoDB działa" || echo "  ❌ MongoDB nie odpowiada"

echo "PostgreSQL:"
docker exec postgresql pg_isready -U admin -d testdb -h localhost 2>/dev/null && echo "  ✅ PostgreSQL działa" || echo "  ❌ PostgreSQL nie odpowiada"

echo "PostgreSQL Exporter:"
curl -s http://localhost:9187/metrics > /dev/null && echo "  ✅ PostgreSQL Exporter działa" || echo "  ❌ PostgreSQL Exporter nie odpowiada"

echo "MongoDB Exporter:"
curl -s http://localhost:9216/metrics > /dev/null && echo "  ✅ MongoDB Exporter działa" || echo "  ❌ MongoDB Exporter nie odpowiada"

sleep 15

echo "✅ Stack został uruchomiony!"
echo ""
echo "📊 Dostępne serwisy:"
echo "   MongoDB:     localhost:27017 (admin/password123)"
echo "   PostgreSQL:  localhost:5432 (admin/password123/testdb)"
echo "   Prometheus:  http://localhost:9090"
echo "   Grafana:     http://localhost:3000 (admin/admin123)"
echo ""
echo "🔧 Następne kroki:"
echo "1. Otwórz Grafana: http://localhost:3000"
echo "2. Zaloguj się (admin/admin123)"
echo "3. Przejdź do Dashboards -> Browse"
echo "4. Dashboardy MongoDB i PostgreSQL powinny być dostępne"
echo ""
echo "💡 Testowanie baz danych:"
echo "   MongoDB:    docker exec -it mongodb mongosh -u admin -p password123 --authenticationDatabase admin"
echo "   PostgreSQL: docker exec -it postgresql psql -U admin -d testdb"
echo ""
echo "📈 Przykładowe zapytania:"
echo "   MongoDB:    use testdb; db.users.find()"
echo "   PostgreSQL: SELECT * FROM users;"
echo ""
echo "🏁 Gotowe! Wszystko powinno działać na localhost."