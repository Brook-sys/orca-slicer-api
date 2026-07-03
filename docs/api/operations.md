# Checklist Operacional

## Antes de Deployar

- [ ] `go test ./...`
- [ ] `go build ./cmd/server`
- [ ] Verificar Dockerfile
- [ ] Verificar `.github/workflows/container.yml`
- [ ] Confirmar que não há secrets em commits
- [ ] Verificar `git status`

## Depois do Deploy

- [ ] Confirmar healthcheck

```bash
curl http://HOST:3000/health
```

- [ ] Confirmar Swagger

```txt
http://HOST:3000/api-docs
```

- [ ] Confirmar profiles

```bash
curl http://HOST:3000/profiles/printers
curl http://HOST:3000/profiles/presets
curl http://HOST:3000/profiles/filaments
```

- [ ] Testar preview

```bash
curl -X POST http://HOST:3000/slice/preview \
  -F file=@model.stl \
  -F printer=... \
  -F preset=... \
  -F filament=...
```

- [ ] Testar slice

```bash
curl -X POST http://HOST:3000/slice \
  -F file=@model.stl \
  -F printer=... \
  -F preset=... \
  -F filament=... \
  -o result.gcode
```

## Slice Assíncrono

```bash
curl -X POST http://HOST:3000/slice/async \
  -F file=@model.stl \
  -F printer=... \
  -F preset=... \
  -F filament=...

curl http://HOST:3000/jobs/JOB_ID
curl -OJ http://HOST:3000/jobs/JOB_ID/download
```

`GET /jobs/{id}` retorna `status` e `progress`. `DELETE /jobs/{id}` cancela um job em andamento.

## Multi-modelos

`POST /slice` e `POST /slice/async` continuam aceitando `file=@model.stl`. Para vários modelos, envie múltiplos campos `files`:

```bash
curl -X POST http://HOST:3000/slice \
  -F files=@part-a.stl \
  -F files=@part-b.stl \
  -F arrange=true \
  -o result.gcode
```

## Profiles Built-in

```bash
curl http://HOST:3000/profiles/builtins/printers
curl http://HOST:3000/profiles/builtins/presets
curl http://HOST:3000/profiles/builtins/filaments
```

## Cache de Resultado

Requests idênticos usam cache em memória com cópia persistida em `DATA_PATH/cache`, evitando apontar para workdirs temporários removidos.

## Variáveis Importantes

| Variável | Valor recomendado |
|----------|-------------------|
| `DATA_PATH` | `/app/data` |
| `ORCASLICER_PATH` | `/app/squashfs-root/AppRun` |
| `ORCA_PROFILES_PATH` | `/app/squashfs-root/resources/profiles` |
| `GENERATE_IMAGE` | `true` se todo G-code precisa thumbnail |
| `RESOLVE_PROFILES` | `true` se todos slices devem resolver inherits por padrão |
| `SANITIZE_PROFILES` | `true` se todos slices devem sanitizar profiles por padrão |
| `SLICE_TIMEOUT_SECONDS` | `1800` ou maior para modelos grandes |

## Verificações de Container

```bash
docker exec orca-slicer-api printenv GENERATE_IMAGE
docker exec orca-slicer-api ls -la /app/data
docker exec orca-slicer-api test -x /app/squashfs-root/AppRun
```

## Quando algo falhar

1. Verificar `/health`
2. Verificar `/slice/debug`
3. Verificar logs do container
4. Testar com `resolveProfiles=true`
5. Testar com `sanitizeProfiles=true`
6. Testar com built-ins conhecidos

## Built-ins Conhecidos Funcionais

```txt
printer: Elegoo Neptune 4 0.4 nozzle
preset: 0.20mm Standard @Elegoo N4 0.4 nozzle
filament: Generic PLA @Elegoo
```
