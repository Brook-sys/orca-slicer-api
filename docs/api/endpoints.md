# Rotas HTTP

Inventário de `cmd/server/main.go`. Categorias de perfis: `printers`, `presets`, `filaments`.

| Método | Rota | Resultado normal |
|---|---|---|
| GET | `/health` | 200 saudável ou 503 degradado |
| GET | `/metrics` | 200, métricas JSON |
| GET | `/openapi.json` | 200, especificação manual |
| GET | `/api-docs` | 200, Swagger UI |
| GET | `/profiles/{category}` | 200, lista de perfis salvos |
| GET | `/profiles/{category}/{name}` | 200, JSON salvo |
| POST | `/profiles/{category}/upload` | 201, grava/substitui perfil |
| POST | `/profiles/{category}/import-url` | 201, importa JSON HTTPS |
| POST | `/profiles/{category}/{name}/update-from-source` | 200, atualização manual |
| DELETE | `/profiles/{category}/{name}` | 204, remove perfil e source |
| GET | `/profiles/builtins/{category}` | 200, lista rasa; pode retornar 404 no layout vendor |
| POST | `/profiles/resolve` | 200, perfil resolvido e warnings |
| POST | `/slice/resolve-profiles` | 200, resolução dos nomes selecionados |
| GET | `/profile-aliases` | 200, aliases persistidos |
| POST | `/profile-aliases` | 200, lista após gravação |
| DELETE | `/profile-aliases/{category}/{from}` | 200, lista após remoção |
| POST | `/slice` | 200, arquivo ou ZIP |
| POST | `/slice/async` | 202, metadados do job |
| POST | `/slice/preview` | 200, JSON de prévia |
| GET | `/slice/status` | 200, estado global persistido |
| GET | `/slice/debug` | 200, último diagnóstico global |
| GET | `/jobs` | 200, jobs conhecidos, sem paginação |
| GET | `/jobs/{id}` | 200, estado do job |
| GET | `/jobs/{id}/download` | 200, primeiro arquivo do job |
| DELETE | `/jobs/{id}` | 204, tentativa de cancelamento, não remoção |

Referências: [perfis](profiles.md), [resolve/aliases](resolve-aliases.md), [slicing/jobs](slicing.md), [erros](errors.md). Nenhuma rota exige credencial atualmente.
