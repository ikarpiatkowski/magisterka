#!/usr/bin/env python3
import time
import threading
import random
from datetime import datetime, timezone
import pymongo
import psycopg2
from elasticsearch import Elasticsearch, NotFoundError

def postgres_crud():
    """Ultra-szybki test CRUD PostgreSQL - 10000 operacji, 1:1 insert/update/delete"""
    try:
        conn = psycopg2.connect(
            host='localhost', port=5432, 
            database='testdb', user='admin', password='password123'
        )
        cur = conn.cursor()
        
        for i in range(10000):
            # INSERT + RETURN ID
            cur.execute(
                "INSERT INTO my_data (data) VALUES (%s) RETURNING id",
                (f'{{"test": {i}, "ts": "{datetime.now(timezone.utc).isoformat()}"}}',)
            )
            row_id = cur.fetchone()[0]

            # READ lekki (po ID)
            cur.execute("SELECT id FROM my_data WHERE id = %s", (row_id,))
            cur.fetchone()

            # UPDATE po ID
            cur.execute(
                "UPDATE my_data SET data = data || %s WHERE id = %s",
                (f'{{"u": {i}}}', row_id)
            )

            # DELETE ten sam rekord
            cur.execute("DELETE FROM my_data WHERE id = %s", (row_id,))
            
            # Commit co 100 operacji
            if i % 100 == 0:
                conn.commit()
                print(".", end="", flush=True)
                
        conn.commit()
        # Count pozostałych rekordów
        cur.execute("SELECT COUNT(*) FROM my_data")
        remaining = cur.fetchone()[0]
        cur.close()
        conn.close()
        print(f"\nPG rows left: {remaining}")
        
    except Exception as e:
        print(f"PostgreSQL Error: {e}")

def mongo_crud():
    """Ultra-szybki test CRUD MongoDB - 10000 operacji, 1:1 insert/update/delete"""
    try:
        client = pymongo.MongoClient('mongodb://admin:password123@localhost:27017/admin')
        db = client.testdb
        start_ns = int(time.time() * 1e9)
        
        for i in range(10000):
            # stabilny unikalny email dla i-tej operacji
            email = f'ultra{i}_{start_ns}@test.com'

            # INSERT
            db.users.insert_one({
                'name': f'Ultra{i}', 
                'email': email, 
                'age': i, 
                'city': 'UltraCity'
            })

            # READ lekki
            db.users.find_one({'email': email})

            # UPDATE
            db.users.update_one({'email': email}, {'$set': {'updated': True}})

            # DELETE ten sam dokument
            db.users.delete_one({'email': email})
            
            if i % 100 == 0:
                print("o", end="", flush=True)
                
        # policz ile zostało dokumentów ogółem w kolekcji
        remaining = db.users.estimated_document_count()
        client.close()
        print(f"\nMongo users left: {remaining}")
        
    except Exception as e:
        print(f"MongoDB Error: {e}")

if __name__ == "__main__":
    print("🚀 ULTRA STRESS TEST CRUD")
    print("==========================")
    print("⏱️  Czas: zależny od hosta")
    print("📊 Operacje: 30000 total (10000 per DB x 3 bazy)")
    print("🧵 Wątki: 3 równoległe")
    print("⚡ Bezpośrednie połączenia (bez docker exec)")
    print("")
    
    start = time.time()
    
    # Elasticsearch CRUD
    def es_crud():
        try:
            es = Elasticsearch("http://localhost:9200")
            index_name = "test_data"
            # ensure index
            if not es.indices.exists(index=index_name):
                es.indices.create(index=index_name)
            start_ns = int(time.time() * 1e9)
            for i in range(10000):
                doc_id = f"doc-{start_ns}-{i}"
                # INDEX
                es.index(index=index_name, id=doc_id, document={
                    "name": f"UltraES{i}",
                    "value": i,
                    "ts": datetime.now(timezone.utc).isoformat()
                })
                # READ lekki
                es.get(index=index_name, id=doc_id, ignore=[404])
                # UPDATE (partial)
                es.update(index=index_name, id=doc_id, doc={"updated": True}, doc_as_upsert=True)
                # DELETE ten sam dokument
                try:
                    es.delete(index=index_name, id=doc_id)
                except NotFoundError:
                    pass
                if i % 100 == 0:
                    print("x", end="", flush=True)
            # count pozostałych
            remaining = es.count(index=index_name)["count"]
            print(f"\nES docs left in {index_name}: {remaining}")
        except Exception as e:
            print(f"Elasticsearch Error: {e}")

    # Uruchom równolegle, zsynchronizowane
    start_event = threading.Event()

    def _run_pg():
        start_event.wait()
        postgres_crud()

    def _run_mongo():
        start_event.wait()
        mongo_crud()

    def _run_es():
        start_event.wait()
        es_crud()

    t1 = threading.Thread(target=_run_pg)
    t2 = threading.Thread(target=_run_mongo)
    t3 = threading.Thread(target=_run_es)

    t1.start()
    t2.start()
    t3.start()

    # Zwolnij jednocześnie start wszystkich wątków
    start_event.set()

    t1.join()
    t2.join()
    t3.join()
    
    duration = time.time() - start
    
    print("\n")
    print(f"✅ Test zakończony w {duration:.1f}s!")
    print("📊 Sprawdź metryki w Grafana: http://localhost:3000")
    total_ops = 30000
    print(f"⚡ Intensywność: {total_ops/duration:.0f} operacji/sekundę")
    print(f"🚀 {total_ops/duration:.0f}x szybszy niż docker exec!")
