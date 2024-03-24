### Ddocker 

Ddocker is a wrap for `github.com/docker/docker`. It provides plain way for work with containers. 
Package allows you use single DockerApiClient for interaction with containers
DockerApiClient's methods: 
  - `ContainersList(ctx context.Context) ([]types.Container, error)`
  - `CreateContainer(ctx context.Context, ctr *containers.Container) (warnings []string, err error)`
  - `ContainerRun(ctx context.Context, ctr *containers.Container) (err error)`
  - `CreateAndRunContainer(ctx context.Context, ctr *containers.Container) error`
  - `ContainerKillAndDelete(ctx context.Context, ctr *containers.Container) (err error)`
  - `ContainerKillAndDeleteAfter(ctx context.Context, ctr *containers.Container, after time.Duration)`
  - `StopContainer(ctx context.Context, ctr *containers.Container) (err error)`
  - `KillContainer(ctx context.Context, ctr *containers.Container) (err error)`

Also it implements interface with two methods:
  - `Init(ctx context.Context) error`
  - `Close() error`

### Predefined containers 

There are two predefined containers in this package: PostgreSQL and RabbitMq based

`PostgreSQL` based container allows you to create Pg database with given  `username`, `password`, `database` and `port`
`RabbitMq` based container allows you to create RabbitMq broker with given default `username` and `password` 


