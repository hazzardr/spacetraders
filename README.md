# spacetraders

## Tools

### APIs
* HTTP framework via `labstack/echo`
* Server codegen via `deepmap/oapi-codegen` from `openapi.yaml`

### Bootstrapping / Config
* CLI via `Cobra`
* Config via `Viper`

### Domain and Storage
* Postgres DB via `pgx`
* DB Migrations via `go-migrate`
* DB Codegen via `sqlc`

```bash
make doctor
make deps # if required
make build
make run
```

# TODO: 
* simple tests
* .env
* docker