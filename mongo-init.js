// Inicjalizacja przykładowej bazy danych z danymi
db = db.getSiblingDB('testdb');

// Tworzenie kolekcji z przykładowymi danymi
db.users.insertMany([
  { name: "Jan Kowalski", email: "jan@example.com", age: 28, city: "Kraków" },
  { name: "Anna Nowak", email: "anna@example.com", age: 32, city: "Warszawa" },
  { name: "Piotr Wiśniewski", email: "piotr@example.com", age: 25, city: "Gdańsk" },
  { name: "Maria Kowalczyk", email: "maria@example.com", age: 29, city: "Wrocław" },
  { name: "Tomasz Lewandowski", email: "tomasz@example.com", age: 35, city: "Poznań" }
]);

db.products.insertMany([
  { name: "Laptop Dell", price: 2500, category: "Electronics", stock: 15 },
  { name: "Smartphone Samsung", price: 1200, category: "Electronics", stock: 32 },
  { name: "Książka 'Wiedźmin'", price: 25, category: "Books", stock: 100 },
  { name: "Kawa arabica", price: 35, category: "Food", stock: 50 },
  { name: "Monitor 4K", price: 800, category: "Electronics", stock: 8 }
]);

// Tworzenie indeksów
db.users.createIndex({ email: 1 }, { unique: true });
db.products.createIndex({ category: 1 });
db.products.createIndex({ price: 1 });

print("Przykładowe dane zostały dodane do bazy testdb");