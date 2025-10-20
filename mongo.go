package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

	opts := options.Client().SetMaxPoolSize(mg.config.Mongo.MaxConnections)

	client, err := mongo.Connect(context.Background(), opts.ApplyURI(uri))
	fail(err, "Unable to create connection pool")

	mg.db = client.Database(mg.config.Mongo.Database)
}
