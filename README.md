# Orca Slicer API

REST API em Go para controlar o OrcaSlicer CLI.

## Desenvolvimento

```bash
go run ./cmd/server
```

## Docker

```bash
docker build -t orca-slicer-api .
docker run --rm -p 3000:3000 orca-slicer-api
```

## Healthcheck

```bash
curl http://localhost:3000/health
```
