package ddocker

import (
	"context"
	"ddocker/consts"
	"ddocker/container"
	"ddocker/container/predefined/postgres"
	"ddocker/container/predefined/postgres/config"
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"log/slog"
	"os"
	"testing"
	"time"
)

var (
	containerName = "testing-container"
	pgImage       = "postgres:latest"
	pgConfig      = config.PostgresContainerConfig{
		Port:     "5432",
		User:     "admin",
		Password: "SomePa55",
		Database: "units",
	}
)

func prepareEngine() *DockerApiClient {
	lg := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	api := NewDockerApi(consts.DOCKER_UNIX_DEFAULT_HOST, consts.DOCKER_DEFAULT_VERSION, lg)
	return api
}

func prepareAndInitEngine() (*DockerApiClient, context.Context, error) {
	api := prepareEngine()
	ctx := context.Background()

	err := api.Init(ctx)
	if err != nil {
		return nil, nil, err
	}

	return api, ctx, nil
}

func TestDockerApiClient_Init(t *testing.T) {
	api := prepareEngine()
	ctx := context.Background()
	defer api.Close()

	err := api.Init(ctx)
	assert.NoError(t, err)
}

// TestDockerApiClient_CreateContainer is an integration test which checks container creation ability
func TestDockerApiClient_CreateContainer(t *testing.T) {
	api, ctx, err := prepareAndInitEngine()
	assert.NoError(t, err)

	defer api.Close()

	ctr := &container.Container{
		Name: containerName,
		Options: container.Options{
			Image: pgImage,
		},
	}

	_, err = api.CreateContainer(ctx, ctr)
	assert.NoError(t, err)
	assert.NotEqual(t, "", ctr.Id)

	list, err := api.ContainersList(ctx)
	assert.NoError(t, err)

	for _, v := range list {
		if v.ID == ctr.Id {
			assert.Equal(t, ctr.Options.Image, v.Image)
			// Because docker's names starts with "/"
			assert.Equal(t, ctr.Name, v.Names[0][1:])
			break
		}
	}

	err = api.ContainerKillAndDelete(ctx, ctr)
	assert.NoError(t, err)
}

// TestDockerApiClient_CreateContainer is an integration test which checks container creation and run ability
func TestDockerApiClient_CreateAndRunContainer(t *testing.T) {
	api, ctx, err := prepareAndInitEngine()
	assert.NoError(t, err)

	defer api.Close()

	ctr := &container.Container{
		Name: containerName,
		Options: container.Options{
			Image: pgImage,
		},
	}

	err = api.CreateAndRunContainer(ctx, ctr)
	assert.NoError(t, err)
	assert.NotEqual(t, "", ctr.Id)

	list, err := api.ContainersList(ctx)
	assert.NoError(t, err)

	for _, v := range list {
		if v.ID == ctr.Id {
			assert.Equal(t, ctr.Options.Image, v.Image)
			// Because docker's names starts with "/"
			assert.Equal(t, ctr.Name, v.Names[0][1:])
			break
		}
	}

	err = api.ContainerKillAndDelete(ctx, ctr)
	assert.NoError(t, err)
}

// TestPredefinedPostgres tests predefines Postgres Container
func TestPredefinedPostgres(t *testing.T) {
	api, ctx, err := prepareAndInitEngine()
	assert.NoError(t, err)

	defer api.Close()

	ctr := postgres.NewPostgresContainer(containerName, pgImage, pgConfig)

	err = api.CreateAndRunContainer(ctx, ctr)
	assert.NoError(t, err)

	time.Sleep(time.Second)

	db, err := sqlx.Connect("postgres", fmt.Sprintf("user=%s dbname=%s password=%s host=localhost port=%s sslmode=disable", pgConfig.User, pgConfig.Database, pgConfig.Password, pgConfig.Port))
	assert.NoError(t, err)

	defer db.Close()

	err = db.Ping()
	assert.Nil(t, err)

	err = api.ContainerKillAndDelete(ctx, ctr)
	assert.NoError(t, err)

	list, err := api.ContainersList(ctx)
	assert.NoError(t, err)

	var exists bool

	for _, v := range list {
		if v.ID == ctr.Id {
			exists = true
			break
		}
	}

	assert.False(t, exists)
}
