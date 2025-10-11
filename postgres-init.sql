CREATE TABLE product (
    id SERIAL PRIMARY KEY,
    jdoc jsonb
);

CREATE INDEX idx__product__price ON product using BTREE(((jdoc -> 'price')::NUMERIC));

INSERT INTO product(jdoc) VALUES ('{"name": "Shampoo", "price": 7.90, "stock": 100}');
INSERT INTO product(jdoc) VALUES ('{"name": "Hairspray", "price": 11.50, "stock": 100}');

UPDATE product SET jdoc = jsonb_set(jdoc, '{stock}', '98') WHERE id = 2;
SELECT id, jdoc->'price' as price, jdoc->'stock' as stock FROM product WHERE (jdoc -> 'price')::numeric < 10;
DELETE FROM product WHERE id = 1;
