package config

type Config struct {
	Server Server `yaml:"server"`
}

type Server struct {
	Address     string   `yaml:"address"`
	Environment string   `yaml:"environment"`
	Database    Database `yaml:"database"`
}

type Database struct {
	Driver string `yaml:"driver"`
	Uri    string `yaml:"uri"`
}
