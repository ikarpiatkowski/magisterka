package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

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
       defer observeLatency(m, "create", now)

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
		// Use bulk enqueue to reduce number of requests
		if p.ElasticsearchId == "" {
			// no id provided - index auto-id
			if id, err := es.EnqueueBulk("index", es.Cfg.IndexName, "", b); err != nil {
				// fallback: synchronous single index
				res, err := es.client.Index(es.Cfg.IndexName, bytes.NewReader(b), es.client.Index.WithContext(es.context))
				if err != nil {
					return fmt.Errorf("index failed after bulk fallback: %w", err)
				}
				defer res.Body.Close()
				if res.IsError() {
					return fmt.Errorf("error indexing document: %s", res.String())
				}
				var r map[string]interface{}
				if err := json.NewDecoder(res.Body).Decode(&r); err == nil {
					if id2, ok := r["_id"].(string); ok {
						p.ElasticsearchId = id2
					}
				}
				return nil
			} else {
				if id != "" {
					p.ElasticsearchId = id
				}
				return nil
			}
		}
		if _, err := es.EnqueueBulk("index", es.Cfg.IndexName, p.ElasticsearchId, b); err != nil {
			// fallback to immediate index
			res, err := es.client.Index(es.Cfg.IndexName, bytes.NewReader(b), es.client.Index.WithDocumentID(p.ElasticsearchId), es.client.Index.WithContext(es.context))
			if err != nil {
				return fmt.Errorf("index failed after bulk fallback: %w", err)
			}
			defer res.Body.Close()
			if res.IsError() {
				return fmt.Errorf("error indexing document: %s", res.String())
			}
			return nil
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}

func (p *product) update(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics) (err error) {
       now := time.Now()
       defer observeLatency(m, "update", now)

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
		// Use Update API with doc to send only changed fields (faster than full reindex)
		doc := map[string]interface{}{"doc": map[string]interface{}{"stock": p.Stock}}
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(doc); err != nil {
			return fmt.Errorf("error encoding update doc: %w", err)
		}
		// Enqueue update using bulk API
		if _, err := es.EnqueueBulk("update", es.Cfg.IndexName, p.ElasticsearchId, buf.Bytes()); err != nil {
			// fallback to single update
			res, err := es.client.Update(es.Cfg.IndexName, p.ElasticsearchId, &buf, es.client.Update.WithContext(es.context), es.client.Update.WithRefresh("false"))
			if err != nil {
				return fmt.Errorf("index (update) failed after bulk fallback: %w", err)
			}
			defer res.Body.Close()
			if res.IsError() {
				return fmt.Errorf("error updating document: %s", res.String())
			}
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}

func (p *product) search(pg *postgres, mg *mongodb, es *elasticsearchStore, db string, m *metrics, debug bool) (err error) {
       now := time.Now()
       defer observeLatency(m, "search", now)

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
			cursor.All(mg.context, &results)
		}
		return nil
	case "es":
		var buf bytes.Buffer
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"range": map[string]interface{}{"price": map[string]interface{}{"lt": 30}},
			},
			"size": 5,
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			return fmt.Errorf("error encoding query: %w", err)
		}
		res, err := es.client.Search(
			es.client.Search.WithContext(es.context),
			es.client.Search.WithIndex(es.Cfg.IndexName),
			es.client.Search.WithBody(&buf),
			es.client.Search.WithSize(5),
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
       defer observeLatency(m, "delete", now)

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
		// Enqueue delete using bulk API
		if _, err := es.EnqueueBulk("delete", es.Cfg.IndexName, p.ElasticsearchId, nil); err != nil {
			// fallback to single delete
			res, err := es.client.Delete(es.Cfg.IndexName, p.ElasticsearchId, es.client.Delete.WithContext(es.context), es.client.Delete.WithRefresh("false"))
			if err != nil {
				return fmt.Errorf("delete failed after bulk fallback: %w", err)
			}
			defer res.Body.Close()
			if res.IsError() {
				return fmt.Errorf("error deleting document: %s", res.String())
			}
		}
		return nil
	}
	return fmt.Errorf("unknown database type: %s", db)
}