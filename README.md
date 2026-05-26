# Job Connect Backend

## Important Setup Step

Before you try to build or run this project from a fresh checkout, generate the Swagger docs:

```bash
swag init -g main.go
```

## Why This Is Required

This repo intentionally does **not** commit generated Swagger files.

The `docs/` folder is ignored in git to avoid frequent merge conflicts from generated OpenAPI output.  
However, `main.go` still imports the generated `job-connect/docs` package so the Swagger UI route can be served.

That means:

- on a fresh clone, the `docs` package does not exist yet
- `go build ./...` and `go run main.go` will fail until Swagger docs are generated

## Typical Local Setup

1. Generate Swagger docs:

```bash
swag init -g main.go
```

2. Build the project:

```bash
go build ./...
```

3. Run the API:

```bash
go run main.go
```

## Swagger UI

After generating docs and starting the server, Swagger is available at:

```text
http://localhost:8080/swagger/index.html
```
