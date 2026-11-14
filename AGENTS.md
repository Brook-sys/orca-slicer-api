# Slicer API Agent Notes

## Stack (real)

- **Go 1.23**, apenas biblioteca padrão. **Zero dependências externas** — não existe
  `go.sum` e não existe motivo para `go mod tidy`.
- Módulo: `github.com/Brook-sys/orca-slicer-api`.
- ⚠️ Este arquivo já afirmou FastAPI/Python/uv/PEP-668. **É falso.** Não há Python,
  pip, uv, venv nem requirements neste projeto. O que existia antes era outra pilha;
  o código atual é Go desde o commit inicial.

## Layout

```
cmd/server/main.go          rotas e wiring
internal/config             variáveis de ambiente (ORCA_VERSION, DATA_PATH, ...)
internal/health             GET /health
internal/metrics            GET /metrics (Prometheus)
internal/jobs               store de jobs assíncronos
internal/profiles           CRUD de perfis em disco (upload/import-url/delete)
internal/slicer              o motor:
  service.go                monta os argumentos do CLI e executa o AppRun
  handler.go                handlers HTTP (sync, async, preview)
  overrides.go              resolve herança de perfis e aplica overrides
  thumbnail.go / metadata.go / state.go / response.go / support.go / types.go
internal/docs               OpenAPI + Swagger UI embutidos no binário
docs/api/*.md               documentação humana (manter em sincronia com as rotas)
Dockerfile                  build multi-stage: binário Go + AppImage do Orca extraído
.github/workflows/          ci.yml, lint.yml, container.yml
```

## Comandos

```bash
go test ./...            # gate principal; tudo passa apenas com stdlib
go build ./cmd/server    # binário
gofmt -l .               # CI não roda gofmt, mas o diff fica sujo sem ele
go vet ./...
```

- **Lint:** o CI usa `golangci-lint` **v1.60 sem `.golangci.yml`** (defaults). Um
  `golangci-lint` v2.x instalado localmente **não é equivalente** — ele exige
  configuração e pode falhar por ausência de `.golangci.yml`. Em dúvida, confie no
  job `lint.yml` do CI em vez do binário local.

## Versão do OrcaSlicer — a armadilha

A versão vive em **dois** lugares e os dois precisam mudar juntos:

| Onde | O quê |
| --- | --- |
| `.github/workflows/container.yml` | `env: ORCA_VERSION:` (usado nos nomes das tags da imagem) |
| `Dockerfile` | `ARG ORCA_VERSION=...` (usado na URL do AppImage, no label e no `ENV`) |

`GET /health` devolve `orcaVersion` **lido da variável de ambiente `ORCA_VERSION`**,
não do binário. Se os dois lugares divergirem, o endpoint **mente**.

Para saber a versão real dentro do contêiner:

```bash
/app/squashfs-root/AppRun --help     # a linha de uso imprime "OrcaSlicer-<versão>"
```

`--version` **não existe** (resposta: `Invalid option --version`). Use o banner de uso.

O nome do asset é derivado da versão:
`OrcaSlicer_Linux_AppImage_Ubuntu2404_V${ORCA_VERSION}.AppImage` — a build quebra se o
upstream (SoftFever/OrcaSlicer) renomear, mover ou apagar o asset. Atualizar a versão
portanto atualiza uma dependência de rede em tempo de build, não só um número.

## Publicação da imagem

- `container.yml`: dispara em push em `main`, tag `v*` e `workflow_dispatch`. Publica
  em `ghcr.io/brook-sys/orca-slicer-api` as tags `main`, `sha-<7>`, `latest`,
  `orca<versão>` e `<tag>-orca<versão>`. **Somente `linux/amd64`.**
- Rollback: as tags `orca<versão>` e `sha-<commit>` já publicadas servem de alvo fixo.
- ⚠️ `ci.yml` tem um job `docker` com `platforms: linux/amd64,linux/arm64`. Ele builda
  arm64, mas o `Dockerfile` **sempre baixa o AppImage amd64** — a imagem arm64 resultado
  não executa em ARM. Buildar sem quebrar dá a ilusão de suporte multi-arch que não
  existe. Para suporte real seria preciso escolher o asset por `TARGETARCH`
  (`..._aarch64_...`, que o upstream publica).

## Como testar uma imagem nova (não há Docker na máquina de desenvolvimento)

A build só acontece no CI. Para exercitar a imagem publicada:

1. suba um contêiner com **volume de dados próprio** e uma porta que não conflita com
   a do stack em produção;
2. `GET /health` — confira `orcaVersion` **e** o banner do AppRun;
3. suba os perfis reais: `POST /profiles/{categoria}/upload` (multipart com `file` e
   `name`) — **o nome do arquivo do `file` precisa terminar em `.json`**, senão o
   servidor responde `400 Invalid file type`;
4. fatie de verdade: `POST /slice` com `file`, `printer`, `preset`, `filament`,
   `resolveProfiles`, `sanitizeProfiles`, `generateImage`. Confira o gcode, os
   metadados (`; filament used [g]`, `; estimated printing time`) e os thumbnails
   injetados (`; thumbnail begin`).

Perfis do usuário e dados de ambiente (host, IP, portas, volumes) **nunca** entram neste
arquivo nem nos docs versionados: vão para o `LOCAL.md` (ignorado pelo git).

## Regras de trabalho

- Commits em português, atômicos, padrão convencional (`feat:`, `fix:`, `docs:`).
- `docs/api/*.md` contêm exemplos com versão do Orca e nomes de perfil: ao mudar de
  versão, revisar os exemplos para não documentar uma versão que não roda.
- Nenhuma dependência externa: se precisar de uma, justificar — hoje o projeto é
  deliberativamente stdlib-only.