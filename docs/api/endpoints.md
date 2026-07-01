# Resumo de Endpoints

## Convenção de Documentação

Cada endpoint detalhado segue este padrão:

1. Método e rota
2. Descrição
3. Parâmetros de path/query/form/body
4. Resposta de sucesso
5. Erros comuns
6. Exemplo `curl`

## Health

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/health` | Verifica status da API, `DATA_PATH` e OrcaSlicer |

## Profiles

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/profiles/{category}` | Lista profiles de uma categoria |
| `GET` | `/profiles/{category}/{name}` | Obtém um profile específico |
| `POST` | `/profiles/{category}/upload` | Faz upload de profile JSON |
| `POST` | `/profiles/{category}/import-url` | Importa profile por URL HTTPS raw |
| `POST` | `/profiles/{category}/{name}/update-from-source` | Atualiza profile usando source URL salva |
| `DELETE` | `/profiles/{category}/{name}` | Remove profile |

Categorias válidas:

- `printers`
- `presets`
- `filaments`

## Aliases

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/profile-aliases` | Lista aliases configurados |
| `POST` | `/profile-aliases` | Cria ou atualiza alias |
| `DELETE` | `/profile-aliases/{category}/{from}` | Remove alias |

## Resolve

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/profiles/resolve` | Resolve um profile com `inherits`, built-ins e overrides |
| `POST` | `/slice/resolve-profiles` | Resolve os profiles de um slicing |

## Slicing

| Método | Rota | Descrição |
|--------|------|-----------|
| `POST` | `/slice` | Executa slicing e retorna G-code/3MF/ZIP |
| `POST` | `/slice/preview` | Gera preview JSON com thumbnail e detecção de suporte. Ver [Preview](preview.md) |
| `GET` | `/slice/status` | Retorna status persistido do último slicing. Ver [Status e Debug](status-debug.md) |
| `GET` | `/slice/debug` | Retorna debug do último slicing. Ver [Status e Debug](status-debug.md) |

## Documentação Interativa

| Método | Rota | Descrição |
|--------|------|-----------|
| `GET` | `/openapi.json` | OpenAPI JSON |
| `GET` | `/api-docs` | Swagger UI |
