# Endpoints

## Health

- `GET /health`

Retorna status do serviço e disponibilidade do OrcaSlicer.

## Profiles

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/profiles/{category}` | Lista profiles de uma categoria |
| GET | `/profiles/{category}/{name}` | Obtém um profile específico |
| POST | `/profiles/{category}/upload` | Upload de profile JSON |
| POST | `/profiles/{category}/import-url` | Importa profile de URL HTTPS raw |
| POST | `/profiles/{category}/{name}/update-from-source` | Atualiza profile a partir da source URL |
| DELETE | `/profiles/{category}/{name}` | Remove profile |

**Categorias válidas**: `printers`, `presets`, `filaments`

## Aliases

| Método | Rota | Descrição |
|--------|------|-----------|
| GET | `/profile-aliases` | Lista aliases |
| POST | `/profile-aliases` | Cria alias |
| DELETE | `/profile-aliases/{category}/{from}` | Remove alias |

## Resolve

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/profiles/resolve` | Resolve um profile com overrides |
| POST | `/slice/resolve-profiles` | Resolve os profiles de um slicing |

## Slicing

| Método | Rota | Descrição |
|--------|------|-----------|
| POST | `/slice` | Executa slicing |
| GET | `/slice/status` | Status do último slicing |
| GET | `/slice/debug` | Debug do último slicing (comando, args, perfis, stdout/stderr) |
| POST | `/slice/preview` | Preview com detecção de suporte (ver seção Preview) |

## Documentação

- `GET /openapi.json` — OpenAPI JSON
- `GET /api-docs` — Swagger UI
