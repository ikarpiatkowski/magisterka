package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8/esapi"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type product struct {
	PostgresId      int      `bson:"-" json:"-"`
	MongoId         string   `bson:"-" json:"-"`
	ElasticsearchId string   `bson:"-" json:"-"`
	Id              any      `bson:"_id,omitempty" json:"id,omitempty"`
	Name            string   `bson:"name,omitempty" json:"name,omitempty"`
	Description     string   `bson:"description,omitempty" json:"description,omitempty"`
	Price           float32  `bson:"price,omitempty" json:"price,omitempty"`
	Stock           int      `bson:"stock,omitempty" json:"stock,omitempty"`
	Colors          []string `bson:"colors,omitempty" json:"colors,omitempty"`
}

func (p *product) create(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		switch db {
case "pg":
			m.createLatency.Observe(time.Since(now).Seconds())
		case "mg":
			m.createLatency.Observe(time.Since(now).Seconds())
		case "es":
			m.createLatency.Observe(time.Since(now).Seconds())
		}
	}()

	switch db {
case "pg":
		b, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("json.Marshal(p) failed: %w", err)
		}
		err = pg.dbpool.QueryRow(pg.context, `INSERT INTO product(jdoc) VALUES ($1) RETURNING id`, b).Scan(&p.PostgresId)
		return annotate(err, "pg.dbpool.QueryRow failed")
	case "mg":
		res, err := mg.db.Collection("product").InsertOne(mg.context, p)
		if err != nil {
			return annotate(err, "InsertOne failed")
		}
		if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
			p.MongoId = oid.Hex()
		}
		return nil
	case "es":
		b, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("json.Marshal failed: %w", err)
		}
		req := esapi.IndexRequest{
			Index:   "products",
			Body:    bytes.NewReader(b),
			Refresh: "true",
		}
		res, err := req.Do(context.Background(), es.client)
		if err != nil {
			return fmt.Errorf("IndexRequest failed: %w", err)
		}
		defer res.Body.Close()

		if res.IsError() {
			return fmt.Errorf("error indexing document: %s", res.String())
		}

		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return fmt.Errorf("error parsing the response body: %s", err)
		}
		p.ElasticsearchId = r["_id"].(string)
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}

func (p *product) update(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		switch db {
case "pg":
			m.updateLatency.Observe(time.Since(now).Seconds())
		case "mg":
			m.updateLatency.Observe(time.Since(now).Seconds())
		case "es":
			m.updateLatency.Observe(time.Since(now).Seconds())
		}
	}()

	switch db {
case "pg":
		_, err = pg.dbpool.Exec(pg.context, `UPDATE product SET jdoc = jsonb_set(jdoc, '{stock}', $1) WHERE id = $2`, p.Stock, p.PostgresId)
		return annotate(err, "pg.dbpool.Exec for update failed")
	case "mg":
		id, err := primitive.ObjectIDFromHex(p.MongoId)
		if err != nil {
			return fmt.Errorf("invalid MongoId: %w", err)
		}
		filter := bson.M{"_id": id}
		update := bson.M{"$set": bson.M{"stock": p.Stock}}
		_, err = mg.db.Collection("product").UpdateOne(mg.context, filter, update)
		return annotate(err, "UpdateOne failed")
	case "es":
		updateDoc := map[string]interface{}{
			"doc": map[string]interface{}{
				"stock": p.Stock,
			},
		}
		b, _ := json.Marshal(updateDoc)
		req := esapi.UpdateRequest{
			Index:      "products",
			DocumentID: p.ElasticsearchId,
			Body:       bytes.NewReader(b),
			Refresh:    "true",
		}
		res, err := req.Do(context.Background(), es.client)
		if err != nil {
			return fmt.Errorf("UpdateRequest failed: %w", err)
		}
		defer res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("error updating document: %s", res.String())
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}

func (p *product) search(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics, debug bool) (err error) {
	now := time.Now()
	defer func() {
		switch db {
case "pg":
			m.searchLatency.Observe(time.Since(now).Seconds())
		case "mg":
			m.searchLatency.Observe(time.Since(now).Seconds())
		case "es":
			m.searchLatency.Observe(time.Since(now).Seconds())
		}
	}()

	switch db {
case "pg":
		rows, err := pg.dbpool.Query(pg.context, `SELECT id, jdoc->'price' as price, jdoc->'stock' as stock FROM product WHERE (jdoc -> 'price')::numeric < $1 LIMIT 5`, 30)
		if err != nil {
			return annotate(err, "pg.dbpool.Query failed")
		}
		defer rows.Close()
		if debug {
			for rows.Next() {
			}
		}
		return nil
	case "mg":
		filter := bson.M{"price": bson.M{"$lt": 30}}
		opts := options.Find().SetLimit(5)
		cursor, err := mg.db.Collection("product").Find(mg.context, filter, opts)
		if err != nil {
			return annotate(err, "Find failed")
		}
		defer cursor.Close(mg.context)
		if debug {
			var results []product
			cursor.All(context.Background(), &results)
		}
		return nil
	case "es":
		var buf bytes.Buffer
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{
					"price": map[string]interface{}{
						"lt": 30,
					},
				},
			},
			"size": 5,
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			return fmt.Errorf("error encoding query: %w", err)
		}
		res, err := es.client.Search(
			es.client.Search.WithContext(context.Background()),
			es.client.Search.WithIndex("products"),
			es.client.Search.WithBody(&buf),
		)
		if err != nil {
			return fmt.Errorf("search request failed: %w", err)
		}
		defer res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("error searching documents: %s", res.String())
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}

func (p *product) delete(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics) (err error) {
	now := time.Now()
	defer func() {
		switch db {
case "pg":
			m.deleteLatency.Observe(time.Since(now).Seconds())
		case "mg":
			m.deleteLatency.Observe(time.Since(now).Seconds())
		case "es":
			m.deleteLatency.Observe(time.Since(now).Seconds())
		}
	}()

	switch db {
case "pg":
		_, err = pg.dbpool.Exec(pg.context, `DELETE FROM product WHERE id = $1`, p.PostgresId)
		return annotate(err, "pg.dbpool.Exec for delete failed")
	case "mg":
		id, err := primitive.ObjectIDFromHex(p.MongoId)
		if err != nil {
			return fmt.Errorf("invalid MongoId: %w", err)
		}
		filter := bson.M{"_id": id}
		_, err = mg.db.Collection("product").DeleteOne(mg.context, filter)
		return annotate(err, "DeleteOne failed")
	case "es":
		req := esapi.DeleteRequest{
			Index:      "products",
			DocumentID: p.ElasticsearchId,
			Refresh:    "true",
		}
		res, err := req.Do(context.Background(), es.client)
		if err != nil {
			return fmt.Errorf("DeleteRequest failed: %w", err)
		}
		defer res.Body.Close()
		if res.IsError() {
			return fmt.Errorf("error deleting document: %s", res.String())
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}