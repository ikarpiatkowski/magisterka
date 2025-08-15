#!/bin/bash

# Kolory dla outputu
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔥 STRESS TEST dla MongoDB i PostgreSQL${NC}"
echo "=================================================="

# Funkcja do generowania losowych danych
generate_test_data() {
    local count=$1
    names=("TestUser" "StressTest" "BenchUser" "LoadTest" "PerfUser")
    cities=("TestCity" "LoadCity" "StressCity" "BenchCity" "PerfCity")
    categories=("TestCat" "LoadCat" "StressCat" "BenchCat" "PerfCat")
    
    name_idx=$((RANDOM % ${#names[@]}))
    city_idx=$((RANDOM % ${#cities[@]}))
    cat_idx=$((RANDOM % ${#categories[@]}))
    
    echo "${names[$name_idx]}_${count}|testuser${count}@stress.com|$((RANDOM % 50 + 18))|${cities[$city_idx]}|TestProduct_${count}|${categories[$cat_idx]}|$((RANDOM % 1000 + 10))|$((RANDOM % 50 + 1))"
}

# Funkcja stress testu dla MongoDB
mongodb_stress_test() {
    local operation=$1
    local iterations=$2
    local concurrent=$3
    
    echo -e "${YELLOW}MongoDB $operation Test - $iterations operacji, $concurrent procesów równoległych${NC}"
    
    case $operation in
        "INSERT")
            # Parallel inserts
            for ((i=1; i<=concurrent; i++)); do
                (
                    start_time=$(date +%s.%N)
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        local_id=$((i * 1000 + j))
                        data=$(generate_test_data $local_id)
                        IFS='|' read -r name email age city product category price stock <<< "$data"
                        
                        docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
                        db = db.getSiblingDB('testdb');
                        db.stress_users.insertOne({
                            name: '$name',
                            email: '$email',
                            age: $age,
                            city: '$city',
                            created_at: new Date(),
                            test_id: $local_id
                        });
                        db.stress_products.insertOne({
                            name: '$product',
                            category: '$category', 
                            price: $price,
                            stock: $stock,
                            test_id: $local_id
                        });" --quiet > /dev/null 2>&1
                        
                        if [ $((j % 50)) -eq 0 ]; then
                            echo "  Proces $i: $j/$(($iterations/$concurrent)) insertów"
                        fi
                    done
                    end_time=$(date +%s.%N)
                    duration=$(echo "$end_time - $start_time" | bc)
                    echo "  Proces $i zakończony w ${duration}s"
                ) &
            done
            wait
            ;;
            
        "SELECT")
            for ((i=1; i<=concurrent; i++)); do
                (
                    start_time=$(date +%s.%N)
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_age=$((RANDOM % 50 + 18))
                        docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
                        db = db.getSiblingDB('testdb');
                        db.stress_users.find({age: {\$gte: $random_age}}).limit(10).toArray();
                        " --quiet > /dev/null 2>&1
                    done
                    end_time=$(date +%s.%N)
                    duration=$(echo "$end_time - $start_time" | bc)
                    echo "  Proces $i (SELECT) zakończony w ${duration}s"
                ) &
            done
            wait
            ;;
            
        "UPDATE")
            for ((i=1; i<=concurrent; i++)); do
                (
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_id=$((RANDOM % 1000 + 1))
                        new_age=$((RANDOM % 50 + 18))
                        docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
                        db = db.getSiblingDB('testdb');
                        db.stress_users.updateMany({test_id: $random_id}, {\$set: {age: $new_age, updated_at: new Date()}});
                        " --quiet > /dev/null 2>&1
                    done
                    echo "  Proces $i (UPDATE) zakończony"
                ) &
            done
            wait
            ;;
            
        "DELETE")
            for ((i=1; i<=concurrent; i++)); do
                (
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_id=$((RANDOM % 500 + 5000))
                        docker exec mongodb mongosh -u admin -p password123 --authenticationDatabase admin --eval "
                        db = db.getSiblingDB('testdb');
                        db.stress_users.deleteMany({test_id: $random_id});
                        " --quiet > /dev/null 2>&1
                    done
                    echo "  Proces $i (DELETE) zakończony"
                ) &
            done
            wait
            ;;
    esac
}

# Funkcja stress testu dla PostgreSQL
postgresql_stress_test() {
    local operation=$1
    local iterations=$2
    local concurrent=$3
    
    echo -e "${YELLOW}PostgreSQL $operation Test - $iterations operacji, $concurrent procesów równoległych${NC}"
    
    case $operation in
        "INSERT")
            for ((i=1; i<=concurrent; i++)); do
                (
                    start_time=$(date +%s.%N)
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        local_id=$((i * 1000 + j))
                        data=$(generate_test_data $local_id)
                        IFS='|' read -r name email age city product category price stock <<< "$data"
                        
                        docker exec postgresql psql -U admin -d testdb -c "
                        INSERT INTO users (name, email, age, city) VALUES ('$name', '$email', $age, '$city');
                        INSERT INTO products (name, price, category, stock) VALUES ('$product', $price, '$category', $stock);
                        " > /dev/null 2>&1
                        
                        if [ $((j % 50)) -eq 0 ]; then
                            echo "  Proces $i: $j/$(($iterations/$concurrent)) insertów"
                        fi
                    done
                    end_time=$(date +%s.%N)
                    duration=$(echo "$end_time - $start_time" | bc)
                    echo "  Proces $i zakończony w ${duration}s"
                ) &
            done
            wait
            ;;
            
        "SELECT")
            for ((i=1; i<=concurrent; i++)); do
                (
                    start_time=$(date +%s.%N)
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_age=$((RANDOM % 50 + 18))
                        docker exec postgresql psql -U admin -d testdb -c "
                        SELECT * FROM users WHERE age >= $random_age LIMIT 10;
                        SELECT * FROM products WHERE price > 100 LIMIT 10;
                        " > /dev/null 2>&1
                    done
                    end_time=$(date +%s.%N)
                    duration=$(echo "$end_time - $start_time" | bc)
                    echo "  Proces $i (SELECT) zakończony w ${duration}s"
                ) &
            done
            wait
            ;;
            
        "UPDATE")
            for ((i=1; i<=concurrent; i++)); do
                (
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_age=$((RANDOM % 50 + 18))
                        docker exec postgresql psql -U admin -d testdb -c "
                        UPDATE users SET age = $random_age WHERE id % 100 = $((RANDOM % 100));
                        " > /dev/null 2>&1
                    done
                    echo "  Proces $i (UPDATE) zakończony"
                ) &
            done
            wait
            ;;
            
        "DELETE")
            for ((i=1; i<=concurrent; i++)); do
                (
                    for ((j=1; j<=iterations/concurrent; j++)); do
                        random_id=$((RANDOM % 100 + 1000))
                        docker exec postgresql psql -U admin -d testdb -c "
                        DELETE FROM users WHERE id > $random_id AND id < $(($random_id + 5));
                        " > /dev/null 2>&1
                    done
                    echo "  Proces $i (DELETE) zakończony"
                ) &
            done
            wait
            ;;
    esac
}

# Menu główne
show_menu() {
    echo ""
    echo -e "${BLUE}Wybierz typ stress testu:${NC}"
    echo "1. Szybki test (100 operacji, 2 procesy)"
    echo "2. Średni test (500 operacji, 5 procesów)"
    echo "3. Intensywny test (1000 operacji, 10 procesów)"
    echo "4. Ekstremalny test (2000 operacji, 20 procesów)"
    echo "5. Test CRUD kompletny"
    echo "6. Test tylko INSERT"
    echo "7. Test tylko SELECT"
    echo "8. Monitoring metrics (pokaż aktualne metryki)"
    echo "9. Wyjście"
    echo ""
    read -p "Wybierz opcję (1-9): " choice
}

# Funkcja pokazująca metryki
show_metrics() {
    echo -e "${GREEN}📊 Aktualne metryki systemowe:${NC}"
    echo ""
    
    echo "🔹 MongoDB Metrics:"
    curl -s http://localhost:9216/metrics | grep -E "(mongodb_up|mongodb_connections|mongodb_opcounters_total)" | head -10
    echo ""
    
    echo "🔹 PostgreSQL Metrics:"
    curl -s http://localhost:9187/metrics | grep -E "(pg_up|pg_stat_database|pg_locks)" | head -10
    echo ""
    
    echo "🔹 Container Stats:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" mongodb postgresql postgresql-exporter mongodb-exporter
}

# Funkcja CRUD test
crud_test() {
    local iterations=$1
    local concurrent=$2
    
    echo -e "${RED}🔥 CRUD STRESS TEST START${NC}"
    echo "Iterations: $iterations, Concurrent: $concurrent"
    echo "=============================================="
    
    start_total=$(date +%s)
    
    echo -e "${GREEN}1/4 INSERT Test${NC}"
    mongodb_stress_test "INSERT" $iterations $concurrent &
    postgresql_stress_test "INSERT" $iterations $concurrent &
    wait
    
    sleep 5
    
    echo -e "${GREEN}2/4 SELECT Test${NC}" 
    mongodb_stress_test "SELECT" $iterations $concurrent &
    postgresql_stress_test "SELECT" $iterations $concurrent &
    wait
    
    sleep 5
    
    echo -e "${GREEN}3/4 UPDATE Test${NC}"
    mongodb_stress_test "UPDATE" $((iterations/2)) $concurrent &
    postgresql_stress_test "UPDATE" $((iterations/2)) $concurrent &
    wait
    
    sleep 5
    
    echo -e "${GREEN}4/4 DELETE Test${NC}"
    mongodb_stress_test "DELETE" $((iterations/4)) $concurrent &
    postgresql_stress_test "DELETE" $((iterations/4)) $concurrent &
    wait
    
    end_total=$(date +%s)
    total_duration=$((end_total - start_total))
    
    echo ""
    echo -e "${RED}🏁 STRESS TEST COMPLETED${NC}"
    echo "Total duration: ${total_duration}s"
    echo "=============================================="
    
    show_metrics
}

# Główna pętla programu
while true; do
    show_menu
    
    case $choice in
        1)
            echo -e "${YELLOW}Uruchamianie szybkiego testu...${NC}"
            crud_test 100 2
            ;;
        2)
            echo -e "${YELLOW}Uruchamianie średniego testu...${NC}"
            crud_test 500 5
            ;;
        3)
            echo -e "${YELLOW}Uruchamianie intensywnego testu...${NC}"
            crud_test 1000 10
            ;;
        4)
            echo -e "${YELLOW}Uruchamianie ekstremalnego testu...${NC}"
            crud_test 2000 20
            ;;
        5)
            read -p "Podaj liczbę iteracji: " iter
            read -p "Podaj liczbę procesów równoległych: " conc
            crud_test $iter $conc
            ;;
        6)
            read -p "Podaj liczbę insertów: " iter
            read -p "Podaj liczbę procesów równoległych: " conc
            echo "INSERT Test - MongoDB & PostgreSQL"
            mongodb_stress_test "INSERT" $iter $conc &
            postgresql_stress_test "INSERT" $iter $conc &
            wait
            show_metrics
            ;;
        7)
            read -p "Podaj liczbę selectów: " iter  
            read -p "Podaj liczbę procesów równoległych: " conc
            echo "SELECT Test - MongoDB & PostgreSQL"
            mongodb_stress_test "SELECT" $iter $conc &
            postgresql_stress_test "SELECT" $iter $conc &
            wait
            show_metrics
            ;;
        8)
            show_metrics
            ;;
        9)
            echo -e "${GREEN}Koniec programu!${NC}"
            exit 0
            ;;
        *)
            echo -e "${RED}Nieprawidłowa opcja!${NC}"
            ;;
    esac
    
    echo ""
    read -p "Naciśnij Enter aby kontynuować..."
done