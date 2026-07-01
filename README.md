# Orca Slicer API

REST API em Go para gerenciar profiles JSON e controlar o OrcaSlicer CLI.

## Documentação

Consulte:

```txt
docs/API.md
```

Com a API rodando:

```txt
http://localhost:3000/api-docs
http://localhost:3000/openapi.json
```

## Recursos Principais

- Profiles em arquivos JSON, sem banco de dados
- Categorias: `printers`, `presets`, `filaments`
- Upload e importação por URL HTTPS raw
- Source tracking e update-from-source
- Overrides temporários (não alteram profiles salvos)
- Preview de profiles resolvidos
- Slicing síncrono com OrcaSlicer oficial
- Apenas 1 slicing por vez (lock global)
- Status e debug persistidos do último slicing
- Healthcheck completo
- CORS configurável
- OpenAPI + Swagger UI
- Docker com OrcaSlicer amd64 embutido
- GitHub Actions para GHCR

## Endpoints Principais

| Endpoint | Descrição |
|----------|-----------|
| `GET /health` | Healthcheck |
| `POST /slice` | Executa slicing |
| `POST /slice/preview` | Preview com thumbnail e detecção de suporte |
| `GET /slice/status` | Status do último slicing |
| `GET /slice/debug` | Debug do último slicing |
| `GET /profiles/{category}` | Lista profiles |
| `POST /profiles/{category}/upload` | Upload de profile |
| `POST /profiles/{category}/import-url` | Importa profile por URL |
| `POST /profiles/resolve` | Resolve profile com overrides |
| `GET /profile-aliases` | Lista aliases |
| `GET /openapi.json` | OpenAPI JSON |
| `GET /api-docs` | Swagger UI |

## Parâmetros Importantes de Slicing

| Campo | Descrição |
|-------|-----------|
| `enableSupport` | Força `enable_support` temporário |
| `brimType` | `true` → `auto_brim`; `false` → `no_brim` |
| `printSequenceByObject` | `true` → `by object`; `false` → `by layer` |
| `generateImage` | Gera thumbnail PNG 160x160 |
| `resolveProfiles` | Resolve `inherits` e built-ins |
| `sanitizeProfiles` | Ajusta campos problemáticos para CLI |

## Execução Local

```bash
export ORCASLICER_PATH=/caminho/para/OrcaSlicer/AppRun
export DATA_PATH=./data
export PORT=3000
export GENERATE_IMAGE=true

go run ./cmd/server
```

## Docker

```bash
docker run --rm \
  -p 3000:3000 \
  -v "$(pwd)/data:/app/data" \
  ghcr.io/brook-sys/orca-slicer-api:latest
```

## Docker Compose (exemplo)

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

## Contribuição

A documentação está em `docs/API.md` e `docs/api/`.

Para trabalhar na API, execute:

```bash
go test ./...
go build ./cmd/server
```
