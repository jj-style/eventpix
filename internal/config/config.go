// defines config for the application
package config

import "github.com/google/wire"

type Config struct {
	Server       *Server       `mapstructure:"server"`
	Database     *Database     `mapstructure:"database"`
	Imagor       *Imagor       `mapstructure:"imagor"`
	Nats         *Nats         `mapstructure:"nats"`
	OauthSecrets *OauthSecrets `mapstructure:"oauth"`
	Cache        *Cache        `mapstructure:"cache"`
}

type Server struct {
	Address           string `mapstructure:"address"`
	Environment       string `mapstructure:"environment"`
	ServerUrl         string `mapstructure:"serverUrl"`
	InternalServerUrl string `mapstructure:"internalServerUrl"`
	SecretKey         string `mapstructure:"secretKey"`
	FormbeeKey        string `mapstructure:"formbeeKey"`
	SingleEventMode   bool   `mapstructure:"singleEventMode"`
	DisableSignups    bool   `mapstructure:"disableSignups"`
}

type Database struct {
	Driver        string `mapstructure:"driver"`
	Uri           string `mapstructure:"uri"`
	EncryptionKey string `mapstructure:"encryptionKey"`
}

type Nats struct {
	Url       string `mapstructure:"url"`
	InProcess bool   `mapstructure:"inProcess"`
}

type Cache struct {
	Mode string `mapstructure:"mode"`
	// Address(es) (comma-separated) of server(s)
	Addr     string `mapstructure:"uri"`
	Username string
	Password string
	Ttl      int64 `mapstructure:"ttl"`
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

func CacheProvider(cfg *Config) *Cache {
	return cfg.Cache
}

var Provider = wire.NewSet(DatabaseProvider, NatsProvider, CacheProvider)
