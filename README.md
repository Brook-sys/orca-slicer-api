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

## Profiles

Categorias: `printers`, `presets`, `filaments`.

```bash
curl http://localhost:3000/profiles/presets
curl -F name=standard -F file=@preset.json http://localhost:3000/profiles/presets/upload
curl -X POST http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{"name":"standard","url":"https://raw.githubusercontent.com/user/repo/main/preset.json"}'
curl http://localhost:3000/profiles/presets/standard
curl -X DELETE http://localhost:3000/profiles/presets/standard
```

## Slice

```bash
curl -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard \
  -F filament=pla \
  -F 'overrides={"preset":{"layer_height":"0.16"}}' \
  http://localhost:3000/slice \
  -o result.gcode
```

Overrides são aplicados em arquivos temporários e não alteram os profiles originais.
