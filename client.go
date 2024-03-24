package ddocker

import (
	"context"
	"errors"
	"github.com/docker/docker/api/types"
	"io"
	"log/slog"
	"os"
	"sync"
	"syscall"
	"time"

	containers "ddocker/container"
	ctr_errors "ddocker/errors"
	"ddocker/utils"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/opencontainers/image-spec/specs-go/v1"
)

type DockerApiClient struct {
	lg      *slog.Logger
	api     *client.Client
	images  []string
	mu      *sync.Mutex
	host    string
	version string
}

func NewDockerApi(host string, version string, lg *slog.Logger) *DockerApiClient {
	return &DockerApiClient{
		host:    host,
		version: version,
		lg:      lg,
		images:  make([]string, 0),
		mu:      &sync.Mutex{},
	}
}

func (d *DockerApiClient) Init(ctx context.Context) error {
	lg := d.lg.With("method", "Init")
	api, err := client.NewClientWithOpts(func(c *client.Client) error {
		ops := []client.Opt{
			client.WithHost(d.host),
			client.WithVersion(d.version),
		}
		for _, op := range ops {
			if err := op(c); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		lg.Error("error during api connection: %v", err)
		return err
	}

	if _, err = api.Ping(ctx); err != nil {
		lg.Error("error during api ping: %v", err)
		return err
	}

	d.api = api

	images, err := d.api.ImageList(ctx, image.ListOptions{All: true})
	if err != nil {
		lg.Error("error during image list retrieving: %v", err)
		return err
	}

	for _, i := range images {
		if len(i.RepoTags) > 0 {
			d.images = append(d.images, i.RepoTags[0])
		}
	}
	lg.Info("Init complete")

	return nil
}

func (d *DockerApiClient) ContainersList(ctx context.Context) ([]types.Container, error) {
	list, err := d.api.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	return list, nil
}

func (d *DockerApiClient) CreateContainer(ctx context.Context, ctr *containers.Container) (warnings []string, err error) {
	lg := d.lg.With("method", "CreateContainer")
	ctx = utils.ContextWithLogger(ctx, lg)

	if ctr.Options.Image == "" {
		return []string{}, ctr_errors.ErrEmptyImageName
	}

	if err = d.verifyImage(ctr.Options.Image); err != nil {
		if !errors.Is(err, ctr_errors.ErrImageNotFound) {
			lg.Error("image verification failed with error: %v", err)
			return []string{}, err
		}

		lg.Info("image not found. Trying to pull ...")

		if err = d.pullImage(ctx, ctr.Options.Image); err != nil {
			lg.Error("image pull failed with error: %v", err)
			return []string{}, err
		}
	}

	containerConfig := &container.Config{
		Image: ctr.Options.Image,
		Env:   ctr.Options.Env,
	}

	ports, err := ctr.Options.NatExposedPorts()
	if err != nil {
		return []string{}, err
	}

	if ports != nil {
		containerConfig.ExposedPorts = ports
	}

	hostConfig := &container.HostConfig{}
	portsMap, err := ctr.Options.PortsMap()
	if err != nil {
		return []string{}, err
	}

	if portsMap != nil {
		hostConfig.PortBindings = portsMap
	}

	if ctr.Options.Network != "" {
		hostConfig.NetworkMode = container.NetworkMode(ctr.Options.Network)
	}

	resp, err := d.api.ContainerCreate(ctx,
		containerConfig,
		hostConfig,
		&network.NetworkingConfig{},
		&v1.Platform{},
		ctr.Name,
	)
	if err != nil {
		return []string{}, err
	}

	ctr.Id = resp.ID

	return resp.Warnings, nil
}

func (d *DockerApiClient) ContainerRun(ctx context.Context, ctr *containers.Container) (err error) {
	lg := d.lg.With("method", "ContainerRun")
	ctx = utils.ContextWithLogger(ctx, lg)

	if err = d.api.ContainerStart(ctx, ctr.Id, container.StartOptions{}); err != nil {
		lg.Error("unable to start container", "error", err)
		return err
	}

	lg.Info("container starts successfully", "id", ctr.Id)

	if ctr.Options.WithListener {
		reader, err := d.api.ContainerLogs(ctx, ctr.Id, container.LogsOptions{
			ShowStderr: true,
			ShowStdout: true,
			Follow:     true,
		})
		if err != nil {
			return err
		}
		go func(rc io.ReadCloser) {
			defer func() {
				if err := rc.Close(); err != nil {
					slog.Default().Error("readCloser close failed", "error", err)
				}
				_, err := io.Copy(os.Stdout, rc)
				if err != nil && err != io.EOF {
					slog.Default().Error("io.Copy failed", "error", err)
				}
			}()
		}(reader)
	}

	if ctr.Options.KillAfter != nil {
		go func(lg *slog.Logger) {
			if err := d.ContainerKillAndDeleteAfter(ctx, ctr, *ctr.Options.KillAfter); err != nil {
				lg.Error("container kill failed", "error", err)
			}
			lg.Info("container killed")
		}(lg)
	}

	return nil
}

func (d *DockerApiClient) CreateAndRunContainer(ctx context.Context, ctr *containers.Container) error {
	lg := d.lg.With("method", "CreateAndRunContainer")
	ctx = utils.ContextWithLogger(ctx, lg)

	warnings, err := d.CreateContainer(ctx, ctr)
	if err != nil {
		return err
	}

	if err != nil {
		lg.Error("container creation failed", "error", err)
		return err
	}

	if len(warnings) > 0 {
		lg.Info("container creation warnings")
		for _, v := range warnings {
			lg.Info(v)
		}
	}

	if err = d.ContainerRun(ctx, ctr); err != nil {
		lg.Error("unable to run container", "error", err)
		return err
	}
	return nil
}

func (d *DockerApiClient) ContainerKillAndDelete(ctx context.Context, ctr *containers.Container) (err error) {
	lg := d.lg.With("method", "ContainerKillAndDelete")
	ctx = utils.ContextWithLogger(ctx, lg)

	if err = d.StopContainer(ctx, ctr); err != nil {
		return err
	}

	if err = d.api.ContainerRemove(ctx, ctr.Id, container.RemoveOptions{}); err != nil {
		return err
	}

	lg.Info("container deleted")

	return nil
}

func (d *DockerApiClient) ContainerKillAndDeleteAfter(ctx context.Context, ctr *containers.Container, after time.Duration) error {
	time.Sleep(after)
	if err := d.ContainerKillAndDelete(ctx, ctr); err != nil {
		return err
	}
	return nil
}

func (d *DockerApiClient) StopContainer(ctx context.Context, ctr *containers.Container) (err error) {
	lg := d.lg.With("method", "StopContainer")

	if err = d.api.ContainerStop(ctx, ctr.Id, container.StopOptions{}); err != nil {
		lg.Error("unable to stop container", "error", err)
		return err
	}
	return nil
}

func (d *DockerApiClient) KillContainer(ctx context.Context, ctr *containers.Container) (err error) {
	lg := d.lg.With("method", "KillContainer")
	if err = d.api.ContainerKill(ctx, ctr.Id, syscall.SIGTERM.String()); err != nil {
		lg.Error("unable to kill container", "error", err)
		return err
	}

	return nil
}

func (d *DockerApiClient) Close() error {
	return d.api.Close()
}

func (d *DockerApiClient) verifyImage(name string) error {
	if !utils.Exists(d.images, name) {
		return ctr_errors.ErrImageNotFound
	}

	return nil
}

func (d *DockerApiClient) pullImage(ctx context.Context, name string) error {
	lg := utils.LoggerFromContext(ctx)

	lg.Info("pulling image ", "name", name)
	readCloser, err := d.api.ImagePull(ctx, name, image.PullOptions{
		All: false,
	})

	if err != nil {
		return err
	}
	defer func() {
		if err := readCloser.Close(); err != nil {
			slog.Default().Error("readCloser closed with error", "error", err)
		}
	}()

	_, err = io.Copy(os.Stdout, readCloser)
	if err != nil && err != io.EOF {
		return err
	}

	lg.Info("pulling complete")

	d.mu.Lock()
	d.images = append(d.images, name)
	d.mu.Unlock()

	return nil
}
