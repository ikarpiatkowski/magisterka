#!/usr/bin/env python3
import time
import threading
import random
from datetime import datetime, timezone
from concurrent.futures import ThreadPoolExecutor, as_completed
import pymongo
import psycopg2
# from elasticsearch import Elasticsearch, NotFoundError

# Parametry konfiguracyjne
TARGET_DURATION_MINUTES = 1  # Czas trwania testu w minutach
NUM_THREADS = 24             # Liczba równoległych wątków dla każdej bazy
DB_HOST = "localhost"        # Host bazy danych

# === Nowe - wybór testowanych baz ===
# Dostępne bazy: 'postgres', 'mongo', 'es'
# Usuń z listy te, których nie chcesz testować.
DATABASES_TO_TEST = ['mongo']

# Słownik do przechowywania połączeń dla każdego wątku
thread_local_connections = threading.local()

# === PostgreSQL ===
def get_postgres_connection():
    """Tworzy i zwraca nowe połączenie do bazy PostgreSQL. Używane w puli wątków."""
    if not hasattr(thread_local_connections, "postgres_conn"):
        thread_local_connections.postgres_conn = psycopg2.connect(
            host=DB_HOST, port=5432,
            database='testdb', user='admin', password='password123'
        )
    return thread_local_connections.postgres_conn

def postgres_crud_task(i):
    """Pojedyncza operacja CRUD dla PostgreSQL."""
    conn = get_postgres_connection()
    try:
        cur = conn.cursor()
        
        # INSERT
        cur.execute(
            "INSERT INTO my_data (data) VALUES (%s) RETURNING id",
            (f'{{"test": {i}, "ts": "{datetime.now(timezone.utc).isoformat()}"}}',)
        )
        row_id = cur.fetchone()[0]

        # SELECT
        cur.execute("SELECT id FROM my_data WHERE id = %s", (row_id,))
        cur.fetchone()

        # UPDATE
        cur.execute(
            "UPDATE my_data SET data = data || %s WHERE id = %s",
            (f'{{"u": {i}}}', row_id)
        )

        # DELETE
        cur.execute("DELETE FROM my_data WHERE id = %s", (row_id,))
        
        conn.commit()
        cur.close()
        return True, None
    except Exception as e:
        return False, e

def _run_postgres_test():
    """Pętla testu PostgreSQL, działająca przez określony czas."""
    print("PostgreSQL: Rozpoczynam test CRUD...")
    start_time = time.time()
    errors = 0
    total_ops = 0
    
    # Wyczyść tabelę przed testem
    with get_postgres_connection() as conn:
        with conn.cursor() as cur:
            cur.execute("TRUNCATE my_data")
            conn.commit()

    with ThreadPoolExecutor(max_workers=NUM_THREADS) as executor:
        futures = set()
        i = 0
        while time.time() - start_time < TARGET_DURATION_MINUTES * 60:
            futures.add(executor.submit(postgres_crud_task, i))
            i += 1
        
        for future in as_completed(futures):
            success, error = future.result()
            if not success:
                errors += 1
            total_ops += 1

    end_time = time.time()
    duration = end_time - start_time
    
    with get_postgres_connection() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT COUNT(*) FROM my_data")
            remaining = cur.fetchone()[0]

    print(f"PostgreSQL: Zakończono test.")
    print(f"Czas trwania: {duration:.2f}s, Operacji: {total_ops}, Błędów: {errors}, Pozostałych wierszy: {remaining}")
    return "postgres", duration, errors, total_ops

# === MongoDB ===
def get_mongo_client():
    """Tworzy i zwraca nowego klienta MongoDB. Używane w puli wątków."""
    if not hasattr(thread_local_connections, "mongo_client"):
        thread_local_connections.mongo_client = pymongo.MongoClient(f'mongodb://admin:password123@{DB_HOST}:27017/admin')
    return thread_local_connections.mongo_client

def mongo_crud_task(i):
    """Pojedyncza operacja CRUD dla MongoDB."""
    client = get_mongo_client()
    try:
        db = client.testdb
        email = f'user{i}_{int(time.time() * 1e9)}@test.com'

        db.users.insert_one({'email': email, 'name': f'User{i}', 'age': i})
        db.users.find_one({'email': email})
        db.users.update_one({'email': email}, {'$set': {'updated': True}})
        db.users.delete_one({'email': email})
        
        return True, None
    except Exception as e:
        return False, e

def _run_mongo_test():
    """Pętla testu MongoDB, działająca przez określony czas."""
    print("MongoDB: Rozpoczynam test CRUD...")
    start_time = time.time()
    errors = 0
    total_ops = 0

    with get_mongo_client() as client:
        client.testdb.users.delete_many({})

    with ThreadPoolExecutor(max_workers=NUM_THREADS) as executor:
        futures = set()
        i = 0
        while time.time() - start_time < TARGET_DURATION_MINUTES * 60:
            futures.add(executor.submit(mongo_crud_task, i))
            i += 1

        for future in as_completed(futures):
            success, error = future.result()
            if not success:
                errors += 1
            total_ops += 1

    end_time = time.time()
    duration = end_time - start_time
    
    with get_mongo_client() as client:
        remaining = client.testdb.users.count_documents({})

    print(f"MongoDB: Zakończono test.")
    print(f"Czas trwania: {duration:.2f}s, Operacji: {total_ops}, Błędów: {errors}, Pozostałych dokumentów: {remaining}")
    return "mongo", duration, errors, total_ops

# === Elasticsearch ===
def get_es_client():
    """Tworzy i zwraca nowego klienta Elasticsearch. Używane w puli wątków."""
    if not hasattr(thread_local_connections, "es_client"):
        thread_local_connections.es_client = Elasticsearch(f"http://{DB_HOST}:9200")
    return thread_local_connections.es_client

def es_crud_task(i):
    """Pojedyncza operacja CRUD dla Elasticsearch."""
    client = get_es_client()
    try:
        index_name = "test_data"
        doc_id = f"doc-{int(time.time() * 1e9)}-{i}"
        
        client.index(index=index_name, id=doc_id, document={"value": i})
        client.get(index=index_name, id=doc_id, ignore=[404])
        client.update(index=index_name, id=doc_id, doc={"updated": True})
        client.delete(index=index_name, id=doc_id)
        
        return True, None
    except Exception as e:
        return False, e

def _run_es_test():
    """Pętla testu Elasticsearch, działająca przez określony czas."""
    print("Elasticsearch: Rozpoczynam test CRUD...")
    start_time = time.time()
    errors = 0
    total_ops = 0
    
    with get_es_client() as client:
        if client.indices.exists(index="test_data"):
            client.indices.delete(index="test_data")
        client.indices.create(index="test_data")

    with ThreadPoolExecutor(max_workers=NUM_THREADS) as executor:
        futures = set()
        i = 0
        while time.time() - start_time < TARGET_DURATION_MINUTES * 60:
            futures.add(executor.submit(es_crud_task, i))
            i += 1
        
        for future in as_completed(futures):
            success, error = future.result()
            if not success:
                errors += 1
            total_ops += 1
    
    end_time = time.time()
    duration = end_time - start_time
    
    with get_es_client() as client:
        remaining = client.count(index="test_data")["count"]

    print(f"Elasticsearch: Zakończono test.")
    print(f"Czas trwania: {duration:.2f}s, Operacji: {total_ops}, Błędów: {errors}, Pozostałych dokumentów: {remaining}")
    return "es", duration, errors, total_ops

if __name__ == "__main__":
    print("🚀 TEST WYDAJNOŚCI CRUD DLA BAZ DANYCH")
    print("======================================")
    print(f"Parametry: {TARGET_DURATION_MINUTES} minut, {NUM_THREADS} wątków równoległych/bazę")
    print("")

    test_functions = {
        'postgres': _run_postgres_test,
        'mongo': _run_mongo_test,
        'es': _run_es_test
    }
    
    selected_tests = [test_functions[db] for db in DATABASES_TO_TEST if db in test_functions]
    
    total_start = time.time()
    
    with ThreadPoolExecutor(max_workers=len(selected_tests)) as executor:
        futures = [executor.submit(test_func) for test_func in selected_tests]
        results = [future.result() for future in futures]
        
    total_duration = time.time() - total_start
    total_ops = sum(res[2] for res in results)
    
    print("\n")
    print("✅ Podsumowanie testów:")
    for db_name, duration, errors, ops in results:
        print(f"{db_name.capitalize()}: {duration:.2f}s, Operacji: {ops}, Błędów: {errors}")
    print("------------------------")
    print(f"Całkowity czas: {total_duration:.2f}s")
    if total_duration > 0:
        print(f"Intensywność: {total_ops / total_duration:.0f} operacji/sekundę")
    print("📊 Sprawdź szczegółowe metryki w Grafana: http://localhost:3000")
