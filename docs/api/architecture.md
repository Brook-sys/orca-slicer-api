# Arquitetura

## Visão Geral

A API é uma camada HTTP em Go que orquestra o OrcaSlicer CLI.

Ela não implementa slicing próprio. O fluxo principal é:

```txt
Cliente HTTP
  -> Handler Go
  -> Service de slicing
  -> Workdir temporário
  -> Profiles temporários
  -> OrcaSlicer CLI
  -> Arquivos gerados
  -> Metadata / Thumbnail / Debug
  -> Resposta HTTP
```

## Componentes

| Componente | Caminho | Responsabilidade |
|------------|---------|------------------|
| Server | `cmd/server/main.go` | Inicializa config, stores, handlers e rotas |
| Config | `internal/config` | Lê variáveis de ambiente |
| HTTPX | `internal/httpx` | Middleware, JSON, erros HTTP |
| Health | `internal/health` | Healthcheck de filesystem e OrcaSlicer |
| Profiles | `internal/profiles` | CRUD/import/update de profiles JSON |
| Slicer | `internal/slicer` | Slicing, preview, debug, resolução e thumbnail |
| Docs | `internal/docs` | OpenAPI e Swagger UI |

## Fluxo de `/slice`

```txt
POST /slice
  -> parse multipart
  -> valida modelo
  -> parse settings
  -> parse profiles multipart opcionais
  -> lock global de slicing
  -> cria workdir temporário
  -> grava modelo e profiles temporários
  -> monta argumentos do OrcaSlicer CLI
  -> executa OrcaSlicer
  -> coleta arquivos gerados
  -> opcionalmente injeta thumbnail
  -> extrai metadata
  -> retorna G-code/3MF/ZIP
```

## Fluxo de `/slice/preview`

```txt
POST /slice/preview
  -> parse multipart
  -> força enableSupport=true
  -> força geração de thumbnail
  -> chama fluxo de slicing
  -> detecta suporte no G-code
  -> extrai thumbnail base64
  -> retorna JSON
```

## Persistência

A API não usa banco de dados.

Tudo fica em `DATA_PATH`:

```txt
DATA_PATH/
  printers/
  presets/
  filaments/
  profile-aliases.json
  slice-status.json
  slice-debug.json
```

## Concorrência

Existe um lock global em memória para garantir apenas 1 slicing por vez.

Motivos:

- OrcaSlicer consome muita RAM/CPU
- Hardware alvo é limitado
- Evita múltiplos processos concorrentes gerando instabilidade

## Segurança

- Upload limitado a 100MB
- Tipos permitidos: STL, STEP, 3MF
- Overrides são temporários
- Profiles salvos não são modificados durante slicing
- Import por URL aceita HTTPS raw JSON

## Debugabilidade

O último job persiste:

- status
- comando
- args
- stdout/stderr
- profiles temporários usados
- result.json do Orca
- erro estruturado do Orca

Endpoints:

```txt
GET /slice/status
GET /slice/debug
```
