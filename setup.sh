#!/bin/bash

echo "🚀 Konfiguracja MongoDB Monitoring Stack"

# Tworzenie struktury katalogów
echo "📁 Tworzenie struktury katalogów..."
mkdir -p grafana/provisioning/datasources
mkdir -p grafana/provisioning/dashboards
mkdir -p grafana/dashboards

# Pobieranie gotowego dashboardu MongoDB z Grafana Labs
echo "📊 Pobieranie oficjalnego dashboardu MongoDB..."
curl -s https://grafana.com/api/dashboards/2583/revisions/2/download > grafana/dashboards/mongodb-dashboard.json

# Alternatywny dashboard - MongoDB Overview
echo "📊 Pobieranie alternatywnego dashboardu MongoDB Overview..."
curl -s https://grafana.com/api/dashboards/14997/revisions/1/download > grafana/dashboards/mongodb-overview.json

# Dodatkowy dashboard - MongoDB Exporter
echo "📊 Pobieranie dashboardu MongoDB Exporter..."
curl -s https://grafana.com/api/dashboards/17016/revisions/1/download > grafana/dashboards/mongodb-exporter.json

# Uruchamianie stacka
echo "🐳 Uruchamianie Docker Compose..."
docker-compose up -d

echo "⏳ Czekanie na uruchomienie wszystkich serwisów..."
sleep 30

echo "✅ Stack został uruchomiony!"
echo ""
echo "📊 Dostępne serwisy:"
echo "   MongoDB:    localhost:27017 (admin/password123)"
echo "   Prometheus: http://localhost:9090"
echo "   Grafana:    http://localhost:3000 (admin/admin123)"
echo ""
echo "🔧 Następne kroki:"
echo "1. Otwórz Grafana: http://localhost:3000"
echo "2. Zaloguj się (admin/admin123)"
echo "3. Przejdź do Dashboards -> MongoDB"
echo "4. Dashboard powinien automatycznie pokazywać metryki"
echo ""
echo "💡 Testowanie MongoDB:"
echo "   docker exec -it mongodb mongosh -u admin -p password123 --authenticationDatabase admin"
echo ""
echo "🏁 Gotowe! Wszystko powinno działać na localhost."