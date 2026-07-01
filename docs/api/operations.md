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

## Variáveis Importantes

| Variável | Valor recomendado |
|----------|-------------------|
| `DATA_PATH` | `/app/data` |
| `ORCASLICER_PATH` | `/app/squashfs-root/AppRun` |
| `ORCA_PROFILES_PATH` | `/app/squashfs-root/resources/profiles` |
| `GENERATE_IMAGE` | `true` se todo G-code precisa thumbnail |
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
