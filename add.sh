#!/bin/bash

echo "🚀 Dodawanie 500 rekordów do MongoDB i PostgreSQL..."

# Funkcja do generowania losowych danych
generate_random_name() {
    names=("Adam" "Anna" "Piotr" "Maria" "Tomasz" "Katarzyna" "Jakub" "Agnieszka" "Łukasz" "Magdalena" "Michał" "Monika" "Paweł" "Joanna" "Krzysztof" "Alicja" "Marcin" "Barbara" "Rafał" "Ewa")
    surnames=("Kowalski" "Nowak" "Wiśniewski" "Wójcik" "Kowalczyk" "Kamiński" "Lewandowski" "Zieliński" "Szymański" "Woźniak" "Dąbrowski" "Kozłowski" "Jankowski" "Mazur" "Kwiatkowski" "Krawczyk" "Kaczmarek" "Piotrowski" "Grabowski" "Nowakowski")
    
    name_idx=$((RANDOM % ${#names[@]}))
    surname_idx=$((RANDOM % ${#surnames[@]}))
    
    echo "${names[$name_idx]} ${surnames[$surname_idx]}"
}

generate_random_city() {
    cities=("Warszawa" "Kraków" "Wrocław" "Poznań" "Gdańsk" "Szczecin" "Bydgoszcz" "Lublin" "Katowice" "Białystok" "Gdynia" "Częstochowa" "Radom" "Sosnowiec" "Toruń" "Kielce" "Gliwice" "Zabrze" "Bytom" "Olsztyn")
    city_idx=$((RANDOM % ${#cities[@]}))
    echo "${cities[$city_idx]}"
}

generate_random_product() {
    products=("Laptop HP" "Smartwatch Apple" "Słuchawki Sony" "Tablet Samsung" "Klawiatura Logitech" "Mysz Razer" "Monitor LG" "Dysk SSD Kingston" "Powerbank Xiaomi" "Kamera GoPro" "Drukarka Canon" "Router TP-Link" "Pendrive SanDisk" "Ładowarka Anker" "Głośnik JBL" "Mikrofon Blue Yeti" "Webcam Logitech" "Hub USB" "Kabel HDMI" "Adapter USB-C")
    categories=("Electronics" "Accessories" "Storage" "Audio" "Video" "Network" "Peripherals")
    
    product_idx=$((RANDOM % ${#products[@]}))
    category_idx=$((RANDOM % ${#categories[@]}))
    
    price=$((RANDOM % 2000 + 50))
    stock=$((RANDOM % 100 + 1))
    
    echo "${products[$product_idx]}|${categories[$category_idx]}|${price}|${stock}"
}

echo "📊 Dodawanie 500 użytkowników do MongoDB..."

# Generowanie danych dla MongoDB
mongodb_users=""
for i in $(seq 1 500); do
    name=$(generate_random_name)
    email="user${i}@example.com"
    age=$((RANDOM % 50 + 18))
    city=$(generate_random_city)
    
    if [ $i -eq 1 ]; then
        mongodb_users="{ name: \"$name\", email: \"$email\", age: $age, city: \"$city\" }"
    else
        mongodb_users="$mongodb_users, { name: \"$name\", email: \"$email\", age: $age, city: \"$city\" }"
    fi
    
    if [ $((i % 100)) -eq 0 ]; then
        echo "  Przygotowano $i/500 użytkowników..."
    fi
done

echo "💾 Wstawianie użytkowników do MongoDB..."
docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
db = db.getSiblingDB('testdb');
db.users.insertMany([$mongodb_users]);
print('Dodano 500 użytkowników do MongoDB');
"

echo "📦 Dodawanie 500 produktów do MongoDB..."

# Generowanie produktów dla MongoDB
mongodb_products=""
for i in $(seq 1 500); do
    product_data=$(generate_random_product)
    IFS='|' read -r name category price stock <<< "$product_data"
    
    if [ $i -eq 1 ]; then
        mongodb_products="{ name: \"$name\", price: $price, category: \"$category\", stock: $stock }"
    else
        mongodb_products="$mongodb_products, { name: \"$name\", price: $price, category: \"$category\", stock: $stock }"
    fi
    
    if [ $((i % 100)) -eq 0 ]; then
        echo "  Przygotowano $i/500 produktów..."
    fi
done

docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
db = db.getSiblingDB('testdb');
db.products.insertMany([$mongodb_products]);
print('Dodano 500 produktów do MongoDB');
"

echo "🐘 Dodawanie 500 użytkowników do PostgreSQL..."

# Generowanie SQL dla PostgreSQL - użytkownicy
postgresql_users_sql="INSERT INTO users (name, email, age, city) VALUES "
for i in $(seq 1 500); do
    name=$(generate_random_name)
    email="pguser${i}@example.com"
    age=$((RANDOM % 50 + 18))
    city=$(generate_random_city)
    
    if [ $i -eq 1 ]; then
        postgresql_users_sql="$postgresql_users_sql('$name', '$email', $age, '$city')"
    else
        postgresql_users_sql="$postgresql_users_sql, ('$name', '$email', $age, '$city')"
    fi
    
    if [ $((i % 100)) -eq 0 ]; then
        echo "  Przygotowano $i/500 użytkowników PostgreSQL..."
    fi
done
postgresql_users_sql="$postgresql_users_sql;"

echo "💾 Wstawianie użytkowników do PostgreSQL..."
docker exec postgresql psql -U admin -d testdb -c "$postgresql_users_sql"

echo "📦 Dodawanie 500 produktów do PostgreSQL..."

# Generowanie SQL dla PostgreSQL - produkty
postgresql_products_sql="INSERT INTO products (name, price, category, stock) VALUES "
for i in $(seq 1 500); do
    product_data=$(generate_random_product)
    IFS='|' read -r name category price stock <<< "$product_data"
    
    if [ $i -eq 1 ]; then
        postgresql_products_sql="$postgresql_products_sql('$name', $price, '$category', $stock)"
    else
        postgresql_products_sql="$postgresql_products_sql, ('$name', $price, '$category', $stock)"
    fi
    
    if [ $((i % 100)) -eq 0 ]; then
        echo "  Przygotowano $i/500 produktów PostgreSQL..."
    fi
done
postgresql_products_sql="$postgresql_products_sql;"

docker exec postgresql psql -U admin -d testdb -c "$postgresql_products_sql"

echo ""
echo "✅ Dodano 500 rekordów do każdej bazy danych!"
echo ""
echo "📊 Sprawdzenie ilości rekordów:"

# Sprawdzenie MongoDB
mongo_user_count=$(docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "db = db.getSiblingDB('testdb'); db.users.countDocuments()" --quiet)
mongo_product_count=$(docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "db = db.getSiblingDB('testdb'); db.products.countDocuments()" --quiet)

echo "MongoDB:"
echo "  Użytkownicy: $mongo_user_count"
echo "  Produkty: $mongo_product_count"

# Sprawdzenie PostgreSQL
pg_user_count=$(docker exec postgresql psql -U admin -d testdb -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')
pg_product_count=$(docker exec postgresql psql -U admin -d testdb -t -c "SELECT COUNT(*) FROM products;" | tr -d ' ')

echo "PostgreSQL:"
echo "  Użytkownicy: $pg_user_count"
echo "  Produkty: $pg_product_count"
echo ""
echo "🎉 Gotowe! Teraz możesz sprawdzić dashboardy w Grafana na http://localhost:3000"