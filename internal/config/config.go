// defines config for the application
package config

import "github.com/google/wire"

type Config struct {
	Server   *Server   `yaml:"server"`
	Database *Database `yaml:"database"`
	Imagor   *Imagor   `yaml:"imagor"`
	Nats     *Nats     `yaml:"nats"`
}

type Server struct {
	Address     string `yaml:"address"`
	Environment string `yaml:"environment"`
	ServerUrl   string `yaml:"serverUrl"`
	SecretKey   string `yaml:"secretKey"`
}

type Database struct {
	Driver string `yaml:"driver"`
	Uri    string `yaml:"uri"`
}

type Nats struct {
	Url       string `yaml:"url"`
	InProcess bool   `yaml:"inProcess"`
}

type Imagor struct {
	Url string `yaml:"url"`
}

func DatabaseProvider(cfg *Config) *Database {
	return cfg.Database
}

func NatsProvider(cfg *Config) *Nats {
	return cfg.Nats
}

var Provider = wire.NewSet(DatabaseProvider, NatsProvider)
