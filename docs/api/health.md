# Healthcheck

## `GET /health`

Verifica disponibilidade da API, filesystem e OrcaSlicer.

## Resposta

```json
{
  "status": "healthy",
  "dataPath": "/app/data",
  "dataPathExists": true,
  "dataPathWritable": true,
  "orcaSlicerPath": "/app/squashfs-root/AppRun",
  "orcaSlicerConfigured": true,
  "orcaSlicerExists": true,
  "orcaSlicerExecutable": true,
  "orcaProfilesPath": "/app/squashfs-root/resources/profiles",
  "orcaProfilesPathExists": true,
  "orcaVersion": "2.4.1",
  "generateImage": true,
  "resolveProfiles": true,
  "sanitizeProfiles": true
}
```

## Campos

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `status` | string | `healthy` ou `unhealthy` |
| `dataPath` | string | Diretório de dados configurado |
| `dataPathExists` | bool | Se o diretório existe |
| `dataPathWritable` | bool | Se a API consegue escrever no diretório |
| `orcaSlicerPath` | string | Caminho do AppRun/binário |
| `orcaSlicerConfigured` | bool | Se `ORCASLICER_PATH` foi configurado |
| `orcaSlicerExists` | bool | Se o binário existe |
| `orcaSlicerExecutable` | bool | Se o binário tem permissão de execução |
| `orcaProfilesPath` | string | Diretório de built-ins do OrcaSlicer |
| `orcaProfilesPathExists` | bool | Se o diretório de built-ins existe |
| `orcaVersion` | string | Versão configurada em `ORCA_VERSION` |
| `generateImage` | bool | Se `GENERATE_IMAGE` está ativo no processo |
| `resolveProfiles` | bool | Se `RESOLVE_PROFILES` está ativo no processo |
| `sanitizeProfiles` | bool | Se `SANITIZE_PROFILES` está ativo no processo |

## Status HTTP

| Status | Significado |
|--------|-------------|
| `200` | API saudável |
| `503` | Alguma verificação essencial falhou |

## Exemplo

```bash
curl http://localhost:3000/health
```
