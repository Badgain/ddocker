package container

import (
	"github.com/docker/go-connections/nat"
	"time"
)

type (
	Container struct {
		Name    string
		Id      string
		Options Options
	}

	Options struct {
		Image        string
		ExposedPorts map[string]string
		Env          []string
		Network      string
		WithListener bool
		KillAfter    *time.Duration
	}
)

func (c *Options) PortsMap() (nat.PortMap, error) {
	if len(c.ExposedPorts) == 0 {
		return nil, nil
	}

	set := make(nat.PortMap)
	for k, v := range c.ExposedPorts {
		ctrPort, err := nat.NewPort("tcp", k)
		if err != nil {
			return nil, err
		}

		dockerPort, err := nat.NewPort("tcp", v)
		if err != nil {
			return nil, err
		}

		set[ctrPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: dockerPort.Port(),
			},
		}
	}

	return set, nil
}

func (c *Options) NatExposedPorts() (nat.PortSet, error) {
	if len(c.ExposedPorts) == 0 {
		return nil, nil
	}

	set := make(nat.PortSet)

	for k := range c.ExposedPorts {
		port, err := nat.NewPort("tcp", k)
		if err != nil {
			return nil, err
		}
		set[port] = struct{}{}
	}

	return set, nil
}
