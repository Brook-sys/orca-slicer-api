# Configuração

## Variáveis de Ambiente

| Variável | Obrigatória | Padrão | Descrição |
|---|---:|---|---|
| `PORT` | Não | `3000` | Porta HTTP |
| `DATA_PATH` | Não | `data` | Diretório de profiles e status |
| `ORCASLICER_PATH` | Sim (slicing) | - | Caminho do AppRun do OrcaSlicer |
| `ORCA_PROFILES_PATH` | Não | `/app/squashfs-root/resources/profiles` | Built-ins para resolver `inherits` |
| `GENERATE_IMAGE` | Não | `false` | Gera thumbnail PNG 160x160 automaticamente |
| `SLICE_TIMEOUT_SECONDS` | Não | `1800` | Timeout máximo de slicing |
| `CORS_ORIGINS` | Não | `*` | Origins separados por vírgula |

## Bind Mount Recomendado

```bash
docker run --rm \
  -p 3000:3000 \
  -v "$(pwd)/data:/app/data" \
  orca-slicer-api
```
