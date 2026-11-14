# Orientações para agentes

## Escopo e fontes de verdade

- Projeto Go 1.23, módulo `github.com/Brook-sys/orca-slicer-api`, somente stdlib. Não há frontend, Python ou banco de dados.
- Leia o código antes de documentar comportamento: rotas em `cmd/server/main.go`, configuração em `internal/config/config.go`, contratos nos handlers e structs.
- Mantenha `README.md`, `docs/API.md`, `docs/api/` e a especificação embutida em `internal/docs/docs.go` coerentes quando o escopo permitir. A especificação é manual, não gerada dos handlers.
- Funcionalidades novas devem ser **planejadas apenas em `docs/ROADMAP.md`**, sem implementação ou promessa de disponibilidade, salvo autorização explícita posterior.

## Mapa do código

| Caminho | Responsabilidade |
|---|---|
| `cmd/server/main.go` | Wiring, rotas, limpeza periódica |
| `internal/config` | Ambiente e defaults |
| `internal/httpx` | CORS, logs e erros JSON |
| `internal/health`, `internal/metrics` | Checks de disco/executável e métricas JSON |
| `internal/profiles` | CRUD, import HTTPS e source tracking |
| `internal/slicer` | CLI, resolução, aliases, cache, thumbnail, estado/debug |
| `internal/jobs` | Jobs em memória e metadados persistidos |
| `internal/docs` | OpenAPI e Swagger UI |
| `Dockerfile` | Go, JSON upstream, AppImage e runtime Ubuntu |
| `.github/workflows` | Testes, lint e imagem |

## Correção 2.5.0 em andamento

Diagnóstico verificado: o AppImage 2.5.0 inspecionado possui 6 JSON e 72 `.opc` internos (cereal); o repositório upstream da mesma tag possui 14.813 JSON. Não tratar `.opc` como ZIP nem implementar conversor especulativo.

O stage `vendorprofiles` obtém JSON da mesma tag e preserva o prefixo `resources/profiles`. A raiz correta é **`/app/profiles-json/resources/profiles`** em `ORCA_PROFILES_PATH`. Não substituir os recursos originais do AppImage.

`loadSelectedProfile` sempre resolve perfis nomeados, mesmo com `resolveProfiles=false`. A resolução aplica herança, overrides e remove `inherits`. Perfis diretos multipart não são achatados automaticamente. BOM UTF-8 é tolerado pelo leitor em disco, não por todas as entradas HTTP. Fallback por tokens equivalentes é último recurso, rejeita ambiguidades e retorna warning nos endpoints de resolução quando usado; revise o resultado. Regressões da atualização ficam em `internal/slicer/overrides_250_test.go`.

O AppImage original com três perfis achatados pela API fatiou cubo e STL de 5.952 triângulos, inclusive este último com arrange=false. Há diferenças de camadas/estimativas frente à referência 2.4.0; não afirmar equivalência ou impressão física validada. A correção foi enviada para build. Isso **não** comprova fluxo completo HTTP, imagem final, publicação concluída ou deploy. Evidências genéricas estão em `docs/api/overview.md`.

## Versão e plataforma

Mantenha alinhados `ARG ORCA_VERSION` no Dockerfile e `env.ORCA_VERSION` em `container.yml`. `/health` informa o valor do ambiente, não detecta a versão. O banner de `AppRun --help` permite conferir o executável; não usar `--version` como check suportado.

O asset usado é amd64 e a imagem publicada é **linux/amd64-only**. `ci.yml` também usa somente amd64, com `push:false`; `container.yml` é o único publicador. Um build Go ARM64 ou manifesto multi-arch não torna o AppImage compatível com ARM. Não anunciar suporte ARM64.

## Validação

```bash
go test ./...
go vet ./...
gofmt -l .
go build -o /tmp/orca-slicer-api ./cmd/server
```

Esses gates não substituem integração com o CLI. Antes de considerar uma atualização validada, use dados sintéticos e volume isolado: health + banner, resolução de herança, slice real com flag true/false, metadados, thumbnails, preview, async/download/cancelamento e erros de concorrência. Veja `docs/api/operations.md`.

Limitações que não podem ser ocultadas: não há autenticação, fila de espera ou isolamento por usuário; cache não cobre todos os inputs; metadados de jobs não recuperam resultados após restart; built-ins não têm listagem recursiva completa; não há limite agregado de upload. `/metrics` retorna JSON, não Prometheus.

## Privacidade e alterações

- Nunca versionar assets reais de teste, perfis pessoais, logs, IPs privados, hostnames, tokens ou detalhes de infraestrutura. Use exemplos genéricos e dados sintéticos; detalhes operacionais ficam em arquivo local ignorado.
- Não apagar nem sobrescrever trabalho existente sem investigar; outras sessões podem estar editando o repositório.
- Respeite o escopo atribuído. Em tarefa somente de documentação, não editar Go, Dockerfile ou workflows.
- Não fazer commit, push, publicação ou deploy sem a autorização aplicável. Publicação não é sinônimo de deploy.
- Antes de entregar, revisar diff, links, contratos HTTP e vazamentos. Não afirmar produção validada com base apenas em testes unitários ou healthcheck.
