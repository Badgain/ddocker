package rabbitmq

import (
	"fmt"
	container2 "github.com/Badgain/ddocker/container"
)

const (
	ImageName           = "rabbitmq:3.13-alpine"
	RabbitMqDefaultPort = "8080"
)

func NewRabbitMqContainer(name string, image string, port string, user string, password string) *container2.Container {
	if image == "" {
		image = ImageName
	}

	if port == "" {
		port = RabbitMqDefaultPort
	}

	return &container2.Container{
		Name: name,
		Options: container2.Options{
			Image: image,
			ExposedPorts: map[string]string{
				port: port,
			},
			Env: []string{
				fmt.Sprintf("RABBITMQ_DEFAULT_USER=%s", user),
				fmt.Sprintf("RABBITMQ_DEFAULT_PASS=%s", password),
			},
		},
	}
}
