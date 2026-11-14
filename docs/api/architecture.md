# Arquitetura

```text
HTTP → handler → validação multipart/JSON → service/lock → arquivos temporários
     → perfis nomeados resolvidos ou JSON diretos → OrcaSlicer CLI
     → coleta de arquivos → thumbnail opcional → metadados → download/preview/job
```

| Pacote | Responsabilidade |
|---|---|
| `cmd/server` | Configuração, dependências, rotas e limpeza periódica |
| `internal/config` | Ambiente e defaults |
| `internal/httpx` | JSON, erros, CORS e logs HTTP |
| `internal/profiles` | Perfis salvos, import/update e checksum/source |
| `internal/slicer` | CLI, resolução/aliases, thumbnails, estado e cache |
| `internal/jobs` | Metadados/progresso/cancelamento async |
| `internal/health`, `internal/metrics` | Checks básicos e métricas JSON |
| `internal/docs` | OpenAPI/Swagger manuais embutidos |

## Estado

```text
DATA_PATH/
  printers/*.json
  presets/*.json
  filaments/*.json
  profile-aliases.json
  slice-status.json
  slice-debug.json
  jobs/*.json
  cache/
```

Perfis importados têm sidecar `.source.json`. Modelos/resultados também usam workdirs temporários fora de `DATA_PATH`; não há coleta completa de todos os casos de erro nem persistência durável de download async.

Um mutex por service permite só um processo de slicing por instância. Não há coordenação entre réplicas; chamadas concorrentes falham, não aguardam. Jobs async usam goroutine/contexto independente do request e o mesmo lock.

## Recursos Orca 2.5.0

O Dockerfile mantém AppImage original em `/app/squashfs-root` e JSON da mesma tag em `/app/profiles-json/resources/profiles`. A API usa os JSON separados para achatar herança; não interpreta caches `.opc`. A resolução por nome é recursiva; a listagem HTTP de built-ins não é e permanece incompatível com a árvore vendor normal.

Preview usa o mesmo pipeline, forçando suporte e thumbnail. Miniatura é renderização STL pela API, não renderização do CLI. Cache é uma otimização com chave incompleta, não fronteira de correção. Consulte [limitações operacionais](operations.md).
