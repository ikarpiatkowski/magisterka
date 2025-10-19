package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Debug bool `yaml:"debug"`

	Elasticsearch ElasticsearchConfig `yaml:"elasticsearch"`
	Postgres      PostgresConfig      `yaml:"postgres"`
	Mongo         MongoConfig         `yaml:"mongo"`

	Test TestConfig `yaml:"test"`
}

type PostgresConfig struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	Host           string `yaml:"host"`
	Database       string `yaml:"database"`
	MaxConnections int    `yaml:"maxConnections"`
	MetricsPort    int    `yaml:"metricsPort"`
}

type MongoConfig struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	Host           string `yaml:"host"`
	Database       string `yaml:"database"`
	MaxConnections uint64 `yaml:"maxConnections"`
	MetricsPort    int    `yaml:"metricsPort"`
}

type ElasticsearchConfig struct {
	Host                    string `yaml:"host"`
	MetricsPort             int    `yaml:"metricsPort"`
	IndexName               string `yaml:"indexName"`
	OpsPerBulk              int    `yaml:"opsPerBulk"`
	WriteProbabilityPercent int    `yaml:"writeProbabilityPercent"`
}

type TestConfig struct {
	MinClients     int `yaml:"minClients"`
	MaxClients     int `yaml:"maxClients"`
	StageIntervalS int `yaml:"stageIntervalS"`
	RequestDelayMs int `yaml:"requestDelayMs"`
}

func (c *Config) loadConfig(path string) {
	f, err := os.ReadFile(path)
	fail(err, "os.ReadFile failed")

	err = yaml.Unmarshal(f, c)
	fail(err, "yaml.Unmarshal failed")
}