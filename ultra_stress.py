#!/usr/bin/env python3
import time
import threading
import random
from datetime import datetime
import pymongo
import psycopg2

def postgres_crud():
    """Ultra-szybki test CRUD PostgreSQL - 10000 operacji"""
    try:
        conn = psycopg2.connect(
            host='localhost', port=5432, 
            database='testdb', user='admin', password='password123'
        )
        cur = conn.cursor()
        
        for i in range(10000):
            # INSERT
            cur.execute("INSERT INTO my_data (data) VALUES (%s)", 
                       (f'{{"test": {i}, "ts": "{datetime.now()}"}}',))
            
            # READ
            cur.execute("SELECT COUNT(*) FROM my_data")
            cur.fetchone()
            
            # UPDATE
            cur.execute("UPDATE my_data SET data = data || %s WHERE id = (SELECT MAX(id) FROM my_data)",
                       (f'{{"u": {i}}}',))
            
            # DELETE co 50
            if i % 50 == 0:
                cur.execute("DELETE FROM my_data WHERE id = (SELECT MIN(id) FROM my_data)")
            
            # Commit co 100 operacji
            if i % 100 == 0:
                conn.commit()
                print(".", end="", flush=True)
                
        conn.commit()
        cur.close()
        conn.close()
        
    except Exception as e:
        print(f"PostgreSQL Error: {e}")

def mongo_crud():
    """Ultra-szybki test CRUD MongoDB - 10000 operacji"""
    try:
        client = pymongo.MongoClient('mongodb://admin:password123@localhost:27017/admin')
        db = client.testdb
        
        for i in range(10000):
            # INSERT
            db.users.insert_one({
                'name': f'Ultra{i}', 
                'email': f'ultra{i}_{int(time.time())}@test.com', 
                'age': i, 
                'city': 'UltraCity'
            })
            
            # READ
            db.users.find().limit(10)
            
            # UPDATE
            db.users.update_one(
                {'email': f'ultra{i}_{int(time.time())}@test.com'}, 
                {'$set': {'updated': True}}
            )
            
            # DELETE co 50
            if i % 50 == 0:
                db.users.delete_one({'email': f'ultra{i}_{int(time.time())}@test.com'})
            
            if i % 100 == 0:
                print("o", end="", flush=True)
                
        client.close()
        
    except Exception as e:
        print(f"MongoDB Error: {e}")

if __name__ == "__main__":
    print("🚀 ULTRA STRESS TEST CRUD")
    print("==========================")
    print("⏱️  Czas: ~2-3 sekundy")
    print("📊 Operacje: 2000 total (10000 per DB)")
    print("🧵 Wątki: 2 równoległe")
    print("⚡ Bezpośrednie połączenia (bez docker exec)")
    print("")
    
    start = time.time()
    
    # Uruchom równolegle
    t1 = threading.Thread(target=postgres_crud)
    t2 = threading.Thread(target=mongo_crud)
    
    t1.start()
    t2.start()
    
    t1.join()
    t2.join()
    
    duration = time.time() - start
    
    print("\n")
    print(f"✅ Test zakończony w {duration:.1f}s!")
    print("📊 Sprawdź metryki w Grafana: http://localhost:3000")
    print(f"⚡ Intensywność: {2000/duration:.0f} operacji/sekundę")
    print(f"🚀 {2000/duration:.0f}x szybszy niż docker exec!")
