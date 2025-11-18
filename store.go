package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	TextContent     string   `bson:"textContent,omitempty" json:"textContent,omitempty"`
}

func (p *product) create(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics) error {
       defer observeLatency(m, "create", time.Now())
       switch db {
       case "pg":
	       b, err := json.Marshal(p)
	       if err != nil {
		       return err
	       }
	       return pg.dbpool.QueryRow(pg.context, `INSERT INTO product(jdoc) VALUES ($1) RETURNING id`, b).Scan(&p.PostgresId)
       case "mg":
	       res, err := mg.db.Collection("product").InsertOne(mg.context, p)
	       if err == nil {
		       if oid, ok := res.InsertedID.(primitive.ObjectID); ok {
			       p.MongoId = oid.Hex()
		       }
	       }
	       return err
       case "es":
	       b, err := json.Marshal(p)
	       if err != nil {
		       return err
	       }
	       id, err := es.EnqueueBulk("index", es.Cfg.IndexName, p.ElasticsearchId, b)
	       if err != nil {
		       res, err := es.client.Index(es.Cfg.IndexName, bytes.NewReader(b), es.client.Index.WithDocumentID(p.ElasticsearchId), es.client.Index.WithContext(es.context))
		       if err != nil {
			       return err
		       }
		       defer res.Body.Close()
		       if !res.IsError() {
			       var r map[string]interface{}
			       if err := json.NewDecoder(res.Body).Decode(&r); err == nil {
				       if id2, ok := r["_id"].(string); ok {
					       p.ElasticsearchId = id2
				       }
			       }
		       }
		       return nil
	       }
	       if id != "" {
		       p.ElasticsearchId = id
	       }
	       return nil
       }
       return nil
}

func (p *product) update(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics) error {
       defer observeLatency(m, "update", time.Now())
       switch db {
       case "pg":
	       _, err := pg.dbpool.Exec(pg.context, `UPDATE product SET jdoc = jsonb_set(jdoc, '{stock}', $1) WHERE id = $2`, p.Stock, p.PostgresId)
	       return err
       case "mg":
	       id, err := primitive.ObjectIDFromHex(p.MongoId)
	       if err != nil {
		       return err
	       }
	       filter := bson.M{"_id": id}
	       update := bson.M{"$set": bson.M{"stock": p.Stock}}
	       _, err = mg.db.Collection("product").UpdateOne(mg.context, filter, update)
	       return err
       case "es":
	       doc := map[string]interface{}{"doc": map[string]interface{}{"stock": p.Stock}}
	       var buf bytes.Buffer
	       if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		       return err
	       }
	       _, err := es.EnqueueBulk("update", es.Cfg.IndexName, p.ElasticsearchId, buf.Bytes())
	       if err != nil {
		       res, err := es.client.Update(es.Cfg.IndexName, p.ElasticsearchId, &buf, es.client.Update.WithContext(es.context), es.client.Update.WithRefresh("false"))
		       if err == nil {
			       defer res.Body.Close()
			       if !res.IsError() {
				       return nil
			       }
		       }
	       }
	       return nil
       }
       return nil
}

func (p *product) search(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics) error {
       defer observeLatency(m, "search", time.Now())
       switch db {
       case "pg":
	       var avg sql.NullFloat64
	       err := pg.dbpool.QueryRow(pg.context, `SELECT AVG(price) FROM (SELECT (jdoc -> 'price')::numeric as price FROM product WHERE (jdoc -> 'price')::numeric < $1 LIMIT 200) as limited_products`, 30).Scan(&avg)
	       if err != nil && err != sql.ErrNoRows {
		       return err
	       }
	       _ = avg
	       return nil
       case "mg":
	       pipeline := []bson.M{
		       {"$match": bson.M{"price": bson.M{"$lt": 30}}},
		       {"$limit": 200},
		       {"$group": bson.M{"_id": nil, "avg_price": bson.M{"$avg": "$price"}}},
	       }
	       cursor, err := mg.db.Collection("product").Aggregate(mg.context, pipeline)
	       if err != nil {
		       return err
	       }
	       defer cursor.Close(mg.context)
	       var out struct{
		       AvgPrice float64 `bson:"avg_price"`
	       }
	       if cursor.Next(mg.context) {
		       if err := cursor.Decode(&out); err != nil {
			       return err
		       }
	       }
	       _ = out
	       return nil
       case "es":
	       var buf bytes.Buffer
	       query := map[string]interface{}{
		       "query": map[string]interface{}{
			       "range": map[string]interface{}{"price": map[string]interface{}{"lt": 30}},
		       },
		       "size": 0,
		       "aggs": map[string]interface{}{
			       "avg_price": map[string]interface{}{"avg": map[string]interface{}{"field": "price"}},
		       },
	       }
	       if err := json.NewEncoder(&buf).Encode(query); err != nil {
		       return err
	       }
	       res, err := es.client.Search(
		       es.client.Search.WithContext(es.context),
		       es.client.Search.WithIndex(es.Cfg.IndexName),
		       es.client.Search.WithBody(&buf),
	       )
	       if err != nil {
		       return err
	       }
	       defer res.Body.Close()
	       var r map[string]interface{}
	       if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		       return err
	       }
	       if aggs, ok := r["aggregations"].(map[string]interface{}); ok {
		       if ap, ok := aggs["avg_price"].(map[string]interface{}); ok {
			       _ = ap["value"]
		       }
	       }
	       return nil
       }
       return nil
}

// NOWA FUNKCJA FTS
func (p *product) searchFTS(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics) error {
	// Używamy nowej etykiety, aby rozróżnić operacje w Grafanie
	defer observeLatency(m, "search_fts", time.Now())
	
	// Wszyscy będą szukać tego samego rzadkiego słowa kluczowego
	keyword := "mongodb" 

	switch db {
	case "pg":
		// Używamy COUNT(*), aby zasymulować agregację, tak jak w starym 'search'
		// Wymaga indeksu: CREATE INDEX idx_product_fts ON product USING GIN (to_tsvector('simple', jdoc ->> 'textContent'));
		var count sql.NullInt64
		err := pg.dbpool.QueryRow(pg.context,
			`SELECT COUNT(*) FROM product 
			 WHERE to_tsvector('simple', jdoc ->> 'textContent') @@ to_tsquery('simple', $1)`,
			keyword).Scan(&count)
		if err != nil && err != sql.ErrNoRows {
			return err
		}
		_ = count // Używamy zmiennej
		return nil

	case "mg":
		// Używamy CountDocuments, aby benchmark był porównywalny
		// Wymaga indeksu: db.product.createIndex({ textContent: "text" })
		filter := bson.M{"$text": bson.M{"$search": keyword}}
		count, err := mg.db.Collection("product").CountDocuments(mg.context, filter)
		if err != nil {
			return err
		}
		_ = count // Używamy zmiennej
		return nil

	case "es":
		// Używamy _count API, które jest zoptymalizowane do tego zadania
		// Wymaga domyślnego indeksu FTS, który Elastic tworzy automatycznie
		var buf bytes.Buffer
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"match": map[string]interface{}{
					"textContent": keyword,
				},
			},
		}
		if err := json.NewEncoder(&buf).Encode(query); err != nil {
			return err
		}

		res, err := es.client.Count(
			es.client.Count.WithContext(es.context),
			es.client.Count.WithIndex(es.Cfg.IndexName),
			es.client.Count.WithBody(&buf),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		// Musimy odczytać ciało odpowiedzi, aby pomiar był kompletny
		var r map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
			return err
		}
		_ = r // Używamy zmiennej
		return nil
	}
	return nil
}

func (p *product) delete(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics) error {
       defer observeLatency(m, "delete", time.Now())
       switch db {
       case "pg":
	       _, err := pg.dbpool.Exec(pg.context, `DELETE FROM product WHERE id = $1`, p.PostgresId)
	       return err
       case "mg":
	       id, err := primitive.ObjectIDFromHex(p.MongoId)
	       if err != nil {
		       return err
	       }
	       filter := bson.M{"_id": id}
	       _, err = mg.db.Collection("product").DeleteOne(mg.context, filter)
	       return err
       case "es":
	       _, err := es.EnqueueBulk("delete", es.Cfg.IndexName, p.ElasticsearchId, nil)
	       if err != nil {
		       res, err := es.client.Delete(es.Cfg.IndexName, p.ElasticsearchId, es.client.Delete.WithContext(es.context), es.client.Delete.WithRefresh("false"))
		       if err == nil {
			       defer res.Body.Close()
			       if !res.IsError() {
				       return nil
			       }
		       }
	       }
	       return nil
       }
       return nil
}