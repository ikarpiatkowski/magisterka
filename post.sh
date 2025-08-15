#!/bin/bash

# Nazwa kontenera i bazy danych
CONTAINER_NAME="postgresql"
DB_NAME="testdb"
DB_USER="admin"
DB_PASSWORD="password123"
POSTGRES_PORT="5432"

# Tworzenie pliku postgres-init.sql
# Ten skrypt zostanie wykonany przy pierwszym uruchomieniu kontenera PostgreSQL
cat <<EOF > postgres-init.sql
CREATE TABLE IF NOT EXISTS my_data (
    id SERIAL PRIMARY KEY,
    data JSONB
);

-- Dodaj indeks GIN na kolumnie 'data' dla efektywnego przeszukiwania JSONB
CREATE INDEX IF NOT EXISTS idx_gin_my_data_data ON my_data USING GIN (data jsonb_path_ops);
EOF

echo "Utworzono plik postgres-init.sql"

# Tworzenie pliku docker-compose.yml
cat <<EOF > docker-compose.yml
version: '3.8'

services:
  ${CONTAINER_NAME}:
    image: postgres:latest
    container_name: ${CONTAINER_NAME}
    restart: unless-stopped
    ports:
      - "${POSTGRES_PORT}:${POSTGRES_PORT}"
    environment:
      POSTGRES_DB: ${DB_NAME}
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - ${CONTAINER_NAME}_data:/var/lib/postgresql/data
      - ./postgres-init.sql:/docker-entrypoint-initdb.d/postgres-init.sql:ro
    networks:
      - monitoring

volumes:
  ${CONTAINER_NAME}_data:

networks:
  monitoring:
    driver: bridge
EOF

echo "Utworzono plik docker-compose.yml"

# Uruchamianie kontenera Docker Compose
echo "Uruchamiam kontener PostgreSQL..."
docker compose up -d

echo "Kontener PostgreSQL uruchomiony w tle. Baza danych '${DB_NAME}' będzie dostępna na porcie ${POSTGRES_PORT}."
echo "Plik postgres-init.sql skonfigurował tabelę 'my_data' z kolumną JSONB i indeksem GIN."