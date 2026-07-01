# Executando

## Localmente

```bash
go run ./cmd/server
```

## Com Docker

```bash
docker run --rm -p 3000:3000 \
  -v "$(pwd)/data:/app/data" \
  ghcr.io/brook-sys/orca-slicer-api:latest
```

## Com Docker Compose (exemplo)

```yaml
services:
  api:
    image: ghcr.io/brook-sys/orca-slicer-api:latest
    ports:
      - "3000:3000"
    volumes:
      - ./data:/app/data
    environment:
      - GENERATE_IMAGE=true
      - SLICE_TIMEOUT_SECONDS=3600
```
