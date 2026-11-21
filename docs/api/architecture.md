# Arquitetura

```text
HTTP → handler → um STL/settings estritos → service/lock → arquivos temporários
     → perfis nomeados resolvidos ou JSON diretos → OrcaSlicer CLI
     → validação de plate_1/manifesto/estatísticas → thumbnail opcional
     → metadados/toolpaths limitados → G-code/preview/job
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

Preview usa o mesmo pipeline e preserva a decisão de suporte. A miniatura é uma renderização separada do STL pela API; os toolpaths são extraídos do G-code com limites duros e truncamento explícito da lista retida. Cache é uma otimização com chave de escopo finito: não captura todo o ambiente do CLI, mas cada hit revalida G-code e evidências. Consulte [limitações operacionais](operations.md).
