package config

type Config struct {
	Server   Server   `yaml:"server"`
	Database Database `yaml:"database"`
	Nats     Nats     `yaml:"nats"`
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
