package postgres

import (
	container2 "ddocker/container"
	"ddocker/container/predefined/postgres/config"
	"fmt"
)

const (
	ImageName = "postgres:latest"
)

func NewPostgresContainer(name string, image string, cfg config.PostgresContainerConfig) *container2.Container {
	if image == "" {
		image = ImageName
	}

	return &container2.Container{
		Name: name,
		Options: container2.Options{
			Image: image,
			ExposedPorts: map[string]string{
				cfg.Port: cfg.Port,
			},
			Env: []string{
				fmt.Sprintf("POSTGRES_DB=%s", cfg.Database),
				fmt.Sprintf("POSTGRES_USER=%s", cfg.User),
				fmt.Sprintf("POSTGRES_PASSWORD=%s", cfg.Password),
			},
		},
	}
}
