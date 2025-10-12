package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type product struct {
	PostgresId  int      `bson:"-" json:"-"` // Ignorujemy w MongoDB i JSON
	MongoId     string   `bson:"-" json:"-"` // Ignorujemy w MongoDB i JSON
	Id          any      `bson:"_id,omitempty" json:"id,omitempty"`
	Name        string   `bson:"name,omitempty" json:"name,omitempty"`
	Description string   `bson:"description,omitempty" json:"description,omitempty"`
	Price       float32  `bson:"price,omitempty" json:"price,omitempty"`
	Stock       int      `bson:"stock,omitempty" json:"stock,omitempty"`
	Colors      []string `bson:"colors,omitempty" json:"colors,omitempty"`
}

func (p *product) create(pg *postgres, mg *mongodb, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		// Niezależnie od błędu, obserwujemy czas trwania
		if db == "pg" {
			m.createLatency.Observe(time.Since(now).Seconds())
		} else {
			m.createLatency.Observe(time.Since(now).Seconds())
		}
	}()

	if db == "pg" {
		b, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("json.Marshal(p) failed: %w", err)
		}
		err = pg.dbpool.QueryRow(pg.context, `INSERT INTO product(jdoc) VALUES ($1) RETURNING id`, b).Scan(&p.PostgresId)
		return annotate(err, "pg.dbpool.QueryRow failed")
	}

	// MongoDB
	res, err := mg.db.Collection("product").InsertOne(mg.context, p)
	if err != nil {
		return annotate(err, "InsertOne failed")
	}
	if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
		p.MongoId = oid.Hex()
	}
	return nil
}

func (p *product) update(pg *postgres, mg *mongodb, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		m.updateLatency.Observe(time.Since(now).Seconds())
	}()

	if db == "pg" {
		_, err = pg.dbpool.Exec(pg.context, `UPDATE product SET jdoc = jsonb_set(jdoc, '{stock}', $1) WHERE id = $2`, p.Stock, p.PostgresId)
		return annotate(err, "pg.dbpool.Exec for update failed")
	}

	// MongoDB
	id, err := primitive.ObjectIDFromHex(p.MongoId)
	if err != nil {
		return fmt.Errorf("invalid MongoId: %w", err)
	}
	filter := bson.M{"_id": id}
	update := bson.M{"$set": bson.M{"stock": p.Stock}}
	_, err = mg.db.Collection("product").UpdateOne(mg.context, filter, update)
	return annotate(err, "UpdateOne failed")
}

func (p *product) search(pg *postgres, mg *mongodb, db string, m *metrics, debug bool) (err error) {
	now := time.Now()
	defer func() {
		m.searchLatency.Observe(time.Since(now).Seconds())
	}()

	if db == "pg" {
		rows, err := pg.dbpool.Query(pg.context, `SELECT id, jdoc->'price' as price, jdoc->'stock' as stock FROM product WHERE (jdoc -> 'price')::numeric < $1 LIMIT 5`, 30)
		if err != nil {
			return annotate(err, "pg.dbpool.Query failed")
		}
		defer rows.Close()

		if debug {
			for rows.Next() {
				var lp product
				if err := rows.Scan(&lp.PostgresId, &lp.Price, &lp.Stock); err != nil {
					return fmt.Errorf("unable to scan row: %w", err)
				}
			}
		}
		return nil
	}

	// MongoDB
	filter := bson.M{"price": bson.M{"$lt": 30}}
	opts := options.Find().SetLimit(5)
	cursor, err := mg.db.Collection("product").Find(mg.context, filter, opts)
	if err != nil {
		return annotate(err, "Find failed")
	}
	defer cursor.Close(mg.context)

	if debug {
		var results []product
		if err = cursor.All(context.TODO(), &results); err != nil {
			return fmt.Errorf("cursor.All failed: %w", err)
		}
	}
	return nil
}

func (p *product) delete(pg *postgres, mg *mongodb, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		m.deleteLatency.Observe(time.Since(now).Seconds())
	}()

	if db == "pg" {
		_, err = pg.dbpool.Exec(pg.context, `DELETE FROM product WHERE id = $1`, p.PostgresId)
		return annotate(err, "pg.dbpool.Exec for delete failed")
	}

	// MongoDB
	id, err := primitive.ObjectIDFromHex(p.MongoId)
	if err != nil {
		return fmt.Errorf("invalid MongoId: %w", err)
	}
	filter := bson.M{"_id": id}
	_, err = mg.db.Collection("product").DeleteOne(mg.context, filter)
	return annotate(err, "DeleteOne failed")
}