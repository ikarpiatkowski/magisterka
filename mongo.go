package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type mongodb struct {
	db     *mongo.Database
	config *Config
	context context.Context
}

func NewMongo(c *Config) *mongodb {
	mg := mongodb{
		config:  c,
		context: context.Background(),
	}
	mg.mgConnect()
	return &mg
}

func (mg *mongodb) mgConnect() {
	var uri string
	if mg.config.Mongo.User != "" && mg.config.Mongo.Password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:27017", mg.config.Mongo.User, mg.config.Mongo.Password, mg.config.Mongo.Host)
	} else {
		uri = fmt.Sprintf("mongodb://%s:27017", mg.config.Mongo.Host)
	}
	// set write concern w=0 (fire-and-forget) on the client
	// use convenience function Unacknowledged() (New and W are deprecated)
	wc := writeconcern.Journaled()
	opts := options.Client().SetMaxPoolSize(mg.config.Mongo.MaxConnections).SetWriteConcern(wc)

	client, err := mongo.Connect(context.Background(), opts.ApplyURI(uri))
	fail(err, "Unable to create connection pool")

	// Also set the database-level write concern to w=0 for operations executed on mg.db
	dbOpts := options.Database().SetWriteConcern(wc)
	mg.db = client.Database(mg.config.Mongo.Database, dbOpts)
}
