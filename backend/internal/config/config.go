package config

import "github.com/google/wire"

type Config struct {
	Server   *Server   `yaml:"server"`
	Database *Database `yaml:"database"`
	PubSub   *PubSub   `yaml:"pubsub"`
}

type Server struct {
	Address     string `yaml:"address"`
	Environment string `yaml:"environment"`
}

type Database struct {
	Driver string `yaml:"driver"`
	Uri    string `yaml:"uri"`
}

type Nats struct {
	Url string `yaml:"url"`
}

type PubSub struct {
	// memory/nats
	Mode string `yaml:"mode"`
	// Whether to run subscribers in process. Assumed true if mode = "memory"
	InProcess bool  `yaml:"inProcess"`
	Workers   uint  `yaml:"workers"`
	Nats      *Nats `yaml:"nats"`
}

func DatabaseProvider(cfg *Config) *Database {
	return cfg.Database
}

func PubSubProvider(cfg *Config) *PubSub {
	return cfg.PubSub
}

var Provider = wire.NewSet(DatabaseProvider, PubSubProvider)
