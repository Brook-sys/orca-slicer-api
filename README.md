# Orca Slicer API

REST API em Go para gerenciar profiles JSON e controlar o OrcaSlicer CLI.

## Recursos principais

- Profiles em arquivos JSON, sem banco de dados.
- Categorias: `printers`, `presets`, `filaments`.
- Upload de profiles via dashboard/API.
- Importação de profiles por URL HTTPS raw JSON.
- Atualização manual de profiles a partir da URL original.
- Overrides temporários sem alterar o profile salvo.
- Preview de profiles resolvidos para dashboard.
- Slicing síncrono com OrcaSlicer.
- Apenas 1 slicing por vez, com lock global.
- Status persistido do último slicing.
- Healthcheck completo.
- CORS configurável.
- OpenAPI e Swagger UI.
- Docker com OrcaSlicer oficial amd64 embutido.
- Publicação de container via GitHub Actions/GHCR.

## Documentação completa

Consulte:

```txt
docs/API.md
```

Com a API rodando:

```txt
http://localhost:3000/api-docs
http://localhost:3000/openapi.json
```

## Execução local

```bash
export ORCASLICER_PATH=/caminho/para/OrcaSlicer
export DATA_PATH=./data
export PORT=3000

go run ./cmd/server
```

## Docker

```bash
docker build --build-arg ORCA_VERSION=2.4.1 -t orca-slicer-api .
mkdir -p ./data
docker run --rm -p 3000:3000 -v "$(pwd)/data:/app/data" orca-slicer-api
```

## Healthcheck

```bash
curl http://localhost:3000/health
```

## Exemplo rápido: upload de profiles

```bash
curl -F name=x1c -F file=@printer.json http://localhost:3000/profiles/printers/upload
curl -F name=standard-020 -F file=@preset.json http://localhost:3000/profiles/presets/upload
curl -F name=pla-basic -F file=@filament.json http://localhost:3000/profiles/filaments/upload
```

## Exemplo rápido: slicing

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -F 'overrides={"preset":{"layer_height":"0.16"}}' \
  -o result.gcode
```

## Testes

```bash
go test ./...
go build ./cmd/server
```
