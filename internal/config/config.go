// defines config for the application
package config

import "github.com/google/wire"

type Config struct {
	Server       *Server       `mapstructure:"server"`
	Database     *Database     `mapstructure:"database"`
	Imagor       *Imagor       `mapstructure:"imagor"`
	Nats         *Nats         `mapstructure:"nats"`
	OauthSecrets *OauthSecrets `mapstructure:"oauth"`
}

type Server struct {
	Address     string `mapstructure:"address"`
	Environment string `mapstructure:"environment"`
	ServerUrl   string `mapstructure:"serverUrl"`
	SecretKey   string `mapstructure:"secretKey"`
	FormbeeKey  string `mapstructure:"formbeeKey"`
}

type Database struct {
	Driver string `mapstructure:"driver"`
	Uri    string `mapstructure:"uri"`
}

type Nats struct {
	Url       string `mapstructure:"url"`
	InProcess bool   `mapstructure:"inProcess"`
}

type Imagor struct {
	Url string `mapstructure:"url"`
}

type OauthSecrets struct {
	Google *GoogleOauth `mapstructure:"google"`
}

type GoogleOauth struct {
	SecretsFile string `mapstructure:"secretsFile"`
	AppId       string `mapstructure:"appId"`
	RedirectUri string `mapstructure:"redirectUri"`
}

func DatabaseProvider(cfg *Config) *Database {
	return cfg.Database
}

func NatsProvider(cfg *Config) *Nats {
	return cfg.Nats
}

var Provider = wire.NewSet(DatabaseProvider, NatsProvider)
