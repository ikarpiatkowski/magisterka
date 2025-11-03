package main

import (
	"bytes"
	"encoding/json"
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

func (p *product) search(pg *postgres, mg *mongodb, es *elastic, db string, m *metrics, debug bool) error {
       defer observeLatency(m, "search", time.Now())
       switch db {
       case "pg":
	       rows, err := pg.dbpool.Query(pg.context, `SELECT id, jdoc->'price' as price, jdoc->'stock' as stock FROM 
		   											product WHERE (jdoc -> 'price')::numeric < $1 LIMIT 5`, 30)
	       if err == nil {
		       defer rows.Close()
		       if debug {
			       for rows.Next() {}
		       }
	       }
	       return err
       case "mg":
	       filter := bson.M{"price": bson.M{"$lt": 30}}
	       opts := options.Find().SetLimit(5)
	       cursor, err := mg.db.Collection("product").Find(mg.context, filter, opts)
	       if err == nil {
		       defer cursor.Close(mg.context)
		       if debug {
			       var results []product
			       cursor.All(mg.context, &results)
		       }
	       }
	       return err
       case "es":
	       var buf bytes.Buffer
	       query := map[string]interface{}{
		       "query": map[string]interface{}{
			       "range": map[string]interface{}{"price": map[string]interface{}{"lt": 30}},
		       },
		       "size": 5,
	       }
	       err := json.NewEncoder(&buf).Encode(query)
	       if err != nil {
		       return err
	       }
	       res, err := es.client.Search(
		       es.client.Search.WithContext(es.context),
		       es.client.Search.WithIndex(es.Cfg.IndexName),
		       es.client.Search.WithBody(&buf),
		       es.client.Search.WithSize(5),
	       )
	       if err == nil {
		       defer res.Body.Close()
	       }
	       return err
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