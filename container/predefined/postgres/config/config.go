package config

type PostgresContainerConfig struct {
	Port     string `json:"port" yaml:"port" default:"5432"`
	User     string `json:"user" yaml:"user" default:"admin"`
	Database string `json:"database" yaml:"database" default:"example"`
	Password string `json:"password" yaml:"password" default:"pa55w0rd"`
}
