# Orca Slicer API - Documentação

API REST em Go para gerenciar profiles JSON do OrcaSlicer e executar slicing usando o OrcaSlicer CLI.

## Sumário

- [Visão geral](#visão-geral)
- [Recursos disponíveis](#recursos-disponíveis)
- [Configuração](#configuração)
- [Executando localmente](#executando-localmente)
- [Executando com Docker](#executando-com-docker)
- [Estrutura de dados](#estrutura-de-dados)
- [Conceitos importantes](#conceitos-importantes)
- [Endpoints](#endpoints)
- [Profiles](#profiles)
- [Overrides](#overrides)
- [Slicing](#slicing)
- [Status do slicing](#status-do-slicing)
- [OpenAPI e Swagger UI](#openapi-e-swagger-ui)
- [Logs](#logs)
- [CORS](#cors)
- [Erros comuns](#erros-comuns)
- [Fluxos recomendados para dashboard](#fluxos-recomendados-para-dashboard)

## Visão geral

Este projeto não implementa um slicer próprio. Ele expõe uma API HTTP para controlar o OrcaSlicer CLI.

Fluxo principal:

```txt
Dashboard/API client
  -> envia modelo + profiles por nome OU JSONs crus no multipart + overrides opcionais
  -> API cria diretório temporário
  -> API grava profiles temporários no workdir
  -> API executa OrcaSlicer CLI
  -> API coleta G-code/3MF gerado
  -> API extrai metadata
  -> API retorna arquivo final
```

Modos de profiles no slicing:

```txt
1. Por nome salvo: printer/preset/filament apontam para DATA_PATH/{category}/{name}.json.
2. Por arquivo multipart: printerProfile/presetProfile/filamentProfile enviam JSONs crus diretamente no /slice.
```

No slicing por nome salvo, a API usa o JSON salvo cru com overrides temporários por padrão.
Se `resolveProfiles=true` for enviado no `/slice`, a API resolve `inherits` e built-ins antes de chamar o OrcaSlicer.
`/profiles/resolve` e `/slice/resolve-profiles` continuam disponíveis para preview/diagnóstico de resolução.

## Recursos disponíveis

Atualmente implementado:

- Healthcheck completo.
- CORS configurável.
- Logs HTTP estruturados.
- CRUD simples de profiles JSON.
- Importação de profile por URL HTTPS raw JSON.
- Source tracking para profiles importados por URL.
- Update manual de profile a partir da source URL original.
- Listagem rica de profiles com metadata.
- Preview de profile resolvido com overrides.
- Preview dos profiles resolvidos para um slicing.
- Slicing síncrono.
- Slicing com profiles salvos por nome.
- Slicing com profiles JSON crus enviados no multipart (`printerProfile`, `presetProfile`, `filamentProfile`).
- Múltiplos `filamentProfile` no mesmo slicing.
- Lock global para permitir apenas 1 slicing por vez.
- Timeout/cancelamento do processo OrcaSlicer.
- Status persistido do último slicing.
- Extração de metadata de G-code/3MF.
- Retorno de arquivo único ou ZIP quando múltiplos arquivos são gerados.
- OpenAPI em `/openapi.json`.
- Swagger UI em `/api-docs`.
- Docker com OrcaSlicer oficial amd64 embutido.
- GitHub Actions para publicar imagem no GHCR.

Não implementado de propósito:

- Slicing assíncrono.
- Autenticação/API key.
- Banco de dados.
- Fila de múltiplos jobs simultâneos.

## Configuração

Variáveis de ambiente:

| Variável | Obrigatória | Padrão | Descrição |
|---|---:|---|---|
| `PORT` | Não | `3000` | Porta HTTP da API. |
| `DATA_PATH` | Não | `data` | Diretório base onde os profiles e status são salvos. |
| `ORCASLICER_PATH` | Sim para slicing | vazio | Caminho absoluto do binário/AppRun do OrcaSlicer. |
| `ORCA_PROFILES_PATH` | Não | `/app/squashfs-root/resources/profiles` | Diretório dos profiles built-in do OrcaSlicer usados para resolver `inherits`. |
| `SLICE_TIMEOUT_SECONDS` | Não | `1800` | Timeout máximo de slicing em segundos. |
| `CORS_ORIGINS` | Não | `*` | Lista de origins separados por vírgula. |

No Docker oficial do projeto:

```txt
ORCASLICER_PATH=/app/squashfs-root/AppRun
ORCA_PROFILES_PATH=/app/squashfs-root/resources/profiles
DATA_PATH=/app/data
PORT=3000
```

## Executando localmente

Com OrcaSlicer instalado localmente:

```bash
export ORCASLICER_PATH=/caminho/para/OrcaSlicer
export DATA_PATH=./data
export PORT=3000

go run ./cmd/server
```

Testar:

```bash
curl http://localhost:3000/health
```

## Executando com Docker

Build local:

```bash
docker build --build-arg ORCA_VERSION=2.4.1 -t orca-slicer-api .
```

Run:

```bash
mkdir -p ./data

docker run --rm \
  -p 3000:3000 \
  -v "$(pwd)/data:/app/data" \
  orca-slicer-api
```

Healthcheck:

```bash
curl http://localhost:3000/health
```

A imagem Docker já inclui as bibliotecas runtime necessárias para o OrcaSlicer AppImage, incluindo OpenGL/GLVND:

```txt
libopengl0
libglu1-mesa
libgl1
libglx0
libegl1
libglvnd0
libsm6
libxext6
libxrender1
```

## Estrutura de dados

A API usa apenas arquivos.

```txt
DATA_PATH/
├── printers/
│   ├── x1c.json
│   └── x1c.source.json
├── presets/
│   ├── standard.json
│   └── standard.source.json
├── filaments/
│   ├── pla.json
│   └── pla.source.json
└── slice-status.json
```

Categorias válidas:

```txt
printers
presets
filaments
```

Nomes válidos para profiles:

```txt
letras, números, underscore e hífen
```

Exemplos válidos:

```txt
x1c
bambu-x1c
pla_basic
standard-020
```

Exemplos inválidos:

```txt
../preset
preset.json
preset com espaço
```

## Conceitos importantes

### Profile salvo

Um profile salvo é um arquivo JSON em uma das pastas:

```txt
DATA_PATH/printers/{name}.json
DATA_PATH/presets/{name}.json
DATA_PATH/filaments/{name}.json
```

### Source tracking

Quando um profile é importado por URL, a API salva um arquivo auxiliar:

```txt
{name}.source.json
```

Exemplo:

```json
{
  "url": "https://raw.githubusercontent.com/user/repo/main/profile.json",
  "checksum": "sha256...",
  "updatedAt": "2026-06-30T00:00:00Z"
}
```

Esse arquivo permite usar `update-from-source` depois.

### Overrides

Overrides são alterações temporárias aplicadas em uma cópia do profile.

O arquivo original nunca é alterado.

Exemplo:

```json
{
  "preset": {
    "layer_height": "0.16",
    "infill_density": "15%"
  }
}
```

Na hora do slicing:

```txt
profile salvo cru OU profile multipart cru + overrides -> profile temporário -> OrcaSlicer
```

Observação: o slicing não resolve `inherits` automaticamente. A resolução de `inherits` fica nos endpoints de preview (`/profiles/resolve` e `/slice/resolve-profiles`) para diagnóstico.

### Lock de slicing

A API permite apenas 1 slicing por vez.

Se outro slicing chegar enquanto um está rodando, a API retorna:

```http
409 Conflict
```

Resposta:

```json
{
  "message": "Slicer is busy"
}
```

## Endpoints

### Docs

```txt
GET /openapi.json
GET /api-docs
```

### Health

```txt
GET /health
```

### Profiles

```txt
GET    /profile-aliases
POST   /profile-aliases
DELETE /profile-aliases/{category}/{from}

GET    /profiles/{category}
GET    /profiles/{category}/{name}
POST   /profiles/{category}/upload
POST   /profiles/{category}/import-url
POST   /profiles/{category}/{name}/update-from-source
DELETE /profiles/{category}/{name}
POST   /profiles/resolve
```

### Slicing

```txt
POST /slice
GET  /slice/status
GET  /slice/debug
POST /slice/resolve-profiles
```

## Healthcheck

### `GET /health`

Verifica:

- status geral;
- se `DATA_PATH` existe;
- se `DATA_PATH` é gravável;
- se `ORCASLICER_PATH` foi configurado;
- se o arquivo do OrcaSlicer existe;
- se é executável.

Exemplo:

```bash
curl http://localhost:3000/health
```

Resposta saudável:

```json
{
  "status": "healthy",
  "dataPathExists": true,
  "dataPathWritable": true,
  "orcaSlicerConfigured": true,
  "orcaSlicerExists": true,
  "orcaSlicerExecutable": true
}
```

Resposta não saudável:

```json
{
  "status": "unhealthy",
  "dataPathExists": true,
  "dataPathWritable": true,
  "orcaSlicerConfigured": false,
  "orcaSlicerExists": false,
  "orcaSlicerExecutable": false
}
```

Status HTTP:

```txt
200 healthy
503 unhealthy
```

## Profiles

### Listar profiles

```txt
GET /profiles/{category}
```

Exemplo:

```bash
curl http://localhost:3000/profiles/presets
```

Resposta:

```json
[
  {
    "name": "standard-020",
    "size": 12345,
    "checksum": "d6e9...",
    "updatedAt": "2026-06-30T10:00:00Z",
    "sourceUrl": "https://raw.githubusercontent.com/user/repo/main/standard.json"
  }
]
```

Campos:

| Campo | Descrição |
|---|---|
| `name` | Nome do profile. |
| `size` | Tamanho do arquivo em bytes. |
| `checksum` | SHA-256 do conteúdo JSON. |
| `updatedAt` | Data de modificação do arquivo local. |
| `sourceUrl` | URL original, se importado via URL. |

### Obter JSON raw de um profile

```txt
GET /profiles/{category}/{name}
```

Exemplo:

```bash
curl http://localhost:3000/profiles/presets/standard-020
```

Resposta:

```json
{
  "layer_height": "0.20"
}
```

### Upload de profile JSON

```txt
POST /profiles/{category}/upload
```

Multipart fields:

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `name` | string | Sim | Nome local do profile. |
| `file` | file `.json` | Sim | Arquivo JSON do OrcaSlicer. |

Exemplo:

```bash
curl -X POST http://localhost:3000/profiles/presets/upload \
  -F name=standard-020 \
  -F file=@standard-020.json
```

Resposta:

```json
{
  "name": "standard-020",
  "checksum": "d6e9...",
  "updated": false
}
```

Observação: upload manual não cria `sourceUrl`.

### Importar profile por URL raw JSON

```txt
POST /profiles/{category}/import-url
```

Body:

```json
{
  "name": "standard-020",
  "url": "https://raw.githubusercontent.com/user/repo/main/standard-020.json",
  "overwrite": false
}
```

Exemplo:

```bash
curl -X POST http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{
    "name":"standard-020",
    "url":"https://raw.githubusercontent.com/user/repo/main/standard-020.json",
    "overwrite":true
  }'
```

Resposta:

```json
{
  "name": "standard-020",
  "checksum": "d6e9...",
  "sourceUrl": "https://raw.githubusercontent.com/user/repo/main/standard-020.json",
  "updated": true
}
```

Regras:

- URL precisa ser HTTPS.
- Conteúdo precisa ser JSON válido.
- Tamanho máximo: 4 MB.
- Se `overwrite=false` e o profile já existir, retorna `409`.

### Atualizar profile pela source URL

```txt
POST /profiles/{category}/{name}/update-from-source
```

Exemplo:

```bash
curl -X POST http://localhost:3000/profiles/presets/standard-020/update-from-source
```

Resposta quando mudou:

```json
{
  "name": "standard-020",
  "checksum": "novo-checksum",
  "sourceUrl": "https://raw.githubusercontent.com/user/repo/main/standard-020.json",
  "updated": true
}
```

Resposta quando não mudou:

```json
{
  "name": "standard-020",
  "checksum": "mesmo-checksum",
  "sourceUrl": "https://raw.githubusercontent.com/user/repo/main/standard-020.json",
  "updated": false
}
```

Se o profile não tiver source URL:

```http
400 Bad Request
```

```json
{
  "message": "Profile has no source URL"
}
```

### Remover profile

```txt
DELETE /profiles/{category}/{name}
```

Exemplo:

```bash
curl -X DELETE http://localhost:3000/profiles/presets/standard-020
```

Resposta:

```txt
204 No Content
```

Também remove o arquivo `.source.json`, se existir.

## Profile aliases

Aliases resolvem diferenças entre nomes de `inherits` e nomes reais dos profiles built-in.

Exemplo real:

```txt
inherits pede: 0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)
built-in existe como: 0.20mm Standard @Elegoo N4 0.4 nozzle
```

A API já possui aliases conhecidos para casos comuns:

```txt
Neptune4 -> N4
Neptune 4 -> N4
Neptune-4 -> N4
```

Também é possível cadastrar aliases customizados em arquivo:

```txt
DATA_PATH/profile-aliases.json
```

### Listar aliases

```txt
GET /profile-aliases
```

```bash
curl http://localhost:3000/profile-aliases
```

### Criar ou atualizar alias

```txt
POST /profile-aliases
```

Body:

```json
{
  "category": "presets",
  "from": "0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)",
  "to": "0.20mm Standard @Elegoo N4 0.4 nozzle"
}
```

Exemplo:

```bash
curl -X POST http://localhost:3000/profile-aliases \
  -H 'Content-Type: application/json' \
  -d '{
    "category":"presets",
    "from":"0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)",
    "to":"0.20mm Standard @Elegoo N4 0.4 nozzle"
  }'
```

### Remover alias

```txt
DELETE /profile-aliases/{category}/{from}
```

Exemplo:

```bash
curl -X DELETE "http://localhost:3000/profile-aliases/presets/0.20mm%20Standard%20%40Elegoo%20Neptune4%20%280.4%20nozzle%29"
```

## Overrides

### Preview de um profile resolvido

```txt
POST /profiles/resolve
```

Body:

```json
{
  "category": "presets",
  "name": "standard-020",
  "overrides": {
    "layer_height": "0.16",
    "infill_density": "15%"
  }
}
```

Exemplo:

```bash
curl -X POST http://localhost:3000/profiles/resolve \
  -H 'Content-Type: application/json' \
  -d '{
    "category":"presets",
    "name":"standard-020",
    "overrides":{"layer_height":"0.16"}
  }'
```

Resposta:

```json
{
  "category": "presets",
  "name": "standard-020",
  "resolved": {
    "layer_height": "0.16",
    "infill_density": "15%"
  },
  "warnings": []
}
```

Se uma chave não existir no profile base, ela ainda é aplicada, mas retorna warning:

```json
{
  "warnings": [
    "Override key does not exist in base profile: unknown_key"
  ]
}
```

### Preview dos profiles resolvidos para slicing

```txt
POST /slice/resolve-profiles
```

Body:

```json
{
  "printer": "x1c",
  "preset": "standard-020",
  "filament": "pla-basic",
  "overrides": {
    "preset": {
      "layer_height": "0.16"
    },
    "filament": {
      "temperature": "220"
    }
  }
}
```

Exemplo:

```bash
curl -X POST http://localhost:3000/slice/resolve-profiles \
  -H 'Content-Type: application/json' \
  -d '{
    "printer":"x1c",
    "preset":"standard-020",
    "filament":"pla-basic",
    "overrides":{
      "preset":{"layer_height":"0.16"}
    }
  }'
```

Resposta:

```json
{
  "printer": {
    "category": "printers",
    "name": "x1c",
    "resolved": {},
    "warnings": []
  },
  "preset": {
    "category": "presets",
    "name": "standard-020",
    "resolved": {
      "layer_height": "0.16"
    },
    "warnings": []
  },
  "filament": {
    "category": "filaments",
    "name": "pla-basic",
    "resolved": {},
    "warnings": []
  }
}
```

## Slicing

### Executar slicing

```txt
POST /slice
```

Multipart fields:

| Campo | Tipo | Obrigatório | Descrição |
|---|---|---:|---|
| `file` | file | Sim | Modelo `.stl`, `.step`, `.stp` ou `.3mf`. |
| `printerProfile` | file `.json` | Não | Profile JSON cru da impressora. Se enviado, tem prioridade sobre `printer`. |
| `presetProfile` | file `.json` | Não | Profile JSON cru do process/preset. Se enviado, tem prioridade sobre `preset`. |
| `filamentProfile` | file `.json` | Não | Profile JSON cru de filamento. Pode ser repetido para múltiplos filamentos. Se enviado, tem prioridade sobre `filament`. |
| `printer` | string | Não | Nome do profile salvo em `DATA_PATH/printers`. Ignorado se `printerProfile` for enviado. |
| `preset` | string | Não | Nome do profile salvo em `DATA_PATH/presets`. Ignorado se `presetProfile` for enviado. |
| `filament` | string | Não | Nome do profile salvo em `DATA_PATH/filaments`. Ignorado se ao menos um `filamentProfile` for enviado. |
| `bedType` | string | Não | Nome do tipo de mesa no OrcaSlicer. |
| `plate` | string | Não | Plate para fatiar. Padrão: `1`. Use `0` para todos. |
| `arrange` | bool | Não | Auto-arrange. |
| `orient` | bool | Não | Auto-orient. |
| `exportType` | string | Não | `gcode` ou `3mf`. Padrão: `gcode`. |
| `multicolorOnePlate` | bool | Não | Ativa `--allow-multicolor-oneplate`. |
| `resolveProfiles` | bool | Não | Quando `true`, resolve `inherits`/built-ins para profiles selecionados por nome antes do slicing. Não afeta `printerProfile`, `presetProfile` ou `filamentProfile` enviados como arquivo. |
| `overrides` | string JSON | Não | JSON com overrides por `printer`, `preset`, `filament`. |

Exemplo básico usando profiles salvos por nome:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -o result.gcode
```

Exemplo usando JSONs crus enviados no próprio request, no mesmo formato usado por sidecars compatíveis com Bambuddy/AFKFelix:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printerProfile=@printer.json \
  -F presetProfile=@preset.json \
  -F filamentProfile=@filament.json \
  -o result.gcode
```

Exemplo com múltiplos filamentos:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.3mf \
  -F printerProfile=@printer.json \
  -F presetProfile=@preset.json \
  -F filamentProfile=@filament-slot-1.json \
  -F filamentProfile=@filament-slot-2.json \
  -F plate=1 \
  -o result.gcode
```

Observações sobre profiles enviados como arquivo:

```txt
- A API grava os JSONs recebidos como arquivos temporários no workdir do slicing.
- A API não resolve inherits desses JSONs.
- A API não busca parents built-in desses JSONs.
- A API não altera compatible_printers desses JSONs.
- A API não persiste esses JSONs em DATA_PATH.
- Overrides ainda podem ser aplicados temporariamente antes de gravar os JSONs temporários.
- printerProfile/presetProfile/filamentProfile têm prioridade sobre printer/preset/filament por nome.
```

Com profiles enviados como arquivo, o comando do OrcaSlicer usa o padrão:

```txt
--slice <plate> --arrange <0|1> --orient <0|1> --load-settings printer.json;preset.json --load-filaments filament_1.json[;filament_2.json]
```

Exemplo com resolução de `inherits`/built-ins antes do slicing:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer='Elegoo Neptune 4 0.4 nozzle' \
  -F preset='0.20mm Standard @Elegoo N4 0.4 nozzle' \
  -F filament='PLA_personalizado1_ONP' \
  -F resolveProfiles=true \
  -o result.gcode
```

Esse modo é útil quando um profile salvo herda de profiles built-in do OrcaSlicer. A API gera JSONs temporários resolvidos apenas para o slicing; os arquivos salvos não são alterados.

Exemplo com overrides:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -F 'overrides={"preset":{"layer_height":"0.16","infill_density":"15%"}}' \
  -o result.gcode
```

Exemplo exportando 3MF:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -F exportType=3mf \
  -o result.3mf
```

Exemplo para todas as plates:

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@project.3mf \
  -F plate=0 \
  -o result.zip
```

### Retorno

Se o OrcaSlicer gerar 1 arquivo:

```txt
application/octet-stream ou tipo detectado pelo http.ServeFile
```

Se gerar múltiplos arquivos:

```txt
application/zip
```

Headers de metadata:

```txt
X-Print-Time-Seconds
X-Filament-Used-g
X-Filament-Used-mm
```

Exemplo:

```bash
curl -i -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -o result.gcode
```

### Argumentos OrcaSlicer gerados

A API monta argumentos parecidos com:

```bash
OrcaSlicer \
  --slice 1 \
  --arrange 0 \
  --orient 0 \
  --load-settings /tmp/slice/input/printer.json\;/tmp/slice/input/preset.json \
  --load-filaments /tmp/slice/input/filament.json \
  --allow-newer-file \
  --outputdir /tmp/slice/output \
  /tmp/slice/input/model.stl
```

Quando `exportType=3mf`:

```txt
--export-3mf result.3mf
```

Quando `bedType` é informado:

```txt
--curr-bed-type {bedType}
```

Quando `multicolorOnePlate=true`:

```txt
--allow-multicolor-oneplate
```

## Status do slicing

### `GET /slice/status`

Retorna o status persistido do último slicing.

Exemplo:

```bash
curl http://localhost:3000/slice/status
```

Resposta idle:

```json
{
  "status": "idle"
}
```

Resposta processando:

```json
{
  "status": "processing",
  "startedAt": "2026-06-30T10:00:00Z"
}
```

Resposta completo:

```json
{
  "status": "completed",
  "startedAt": "2026-06-30T10:00:00Z",
  "finishedAt": "2026-06-30T10:01:00Z",
  "files": ["/tmp/slice-xxx/output/result.gcode"],
  "metadata": {
    "printTime": 3723,
    "filamentUsedG": 12.3,
    "filamentUsedMm": 1234.5
  }
}
```

Resposta com erro:

```json
{
  "status": "failed",
  "startedAt": "2026-06-30T10:00:00Z",
  "finishedAt": "2026-06-30T10:00:10Z",
  "errorMessage": "Slicing failed: ..."
}
```

Possíveis status:

```txt
idle
processing
completed
failed
cancelled
```

Observação: este endpoint não é async job. Ele apenas mostra o último estado conhecido e persiste em `DATA_PATH/slice-status.json`.

### `GET /slice/debug`

Retorna o debug persistido do último slicing, incluindo comando, argumentos, workdir, profiles temporários, stdout/stderr, `result.json` e erro do OrcaSlicer.

Exemplo:

```bash
curl http://localhost:3000/slice/debug
```

Resposta resumida:

```json
{
  "workdir": "/tmp/slice-xxx",
  "command": "/app/squashfs-root/AppRun",
  "args": ["--slice", "1", "--load-settings", "..."],
  "printer": {},
  "preset": {},
  "filament": {},
  "slicerError": "The selected printer is not compatible...",
  "resultJson": {}
}
```

Persiste em `DATA_PATH/slice-debug.json`.

## OpenAPI e Swagger UI

OpenAPI JSON:

```bash
curl http://localhost:3000/openapi.json
```

Swagger UI:

```txt
http://localhost:3000/api-docs
```

## Logs

A API registra logs estruturados com `slog`.

Exemplo de request HTTP:

```txt
level=INFO msg="http request" method=POST path=/slice status=200 bytes=1234 duration_ms=1020
```

Exemplo de slicing:

```txt
level=INFO msg="slicing started" file=model.stl export_type=gcode
level=INFO msg="slicing completed" duration_ms=1020 files=1 print_time_seconds=3723
```

## CORS

Por padrão, CORS aceita todos:

```txt
CORS_ORIGINS=*
```

Para restringir:

```bash
export CORS_ORIGINS="https://dashboard.example.com,https://admin.example.com"
```

Métodos permitidos:

```txt
GET, POST, DELETE, OPTIONS
```

Headers expostos:

```txt
Content-Disposition
ETag
Last-Modified
Content-Length
X-Filament-Used-g
X-Filament-Used-mm
X-Print-Time-Seconds
```

## Erros comuns

### OrcaSlicer não configurado

```json
{
  "message": "ORCASLICER_PATH is not configured"
}
```

Solução:

```bash
export ORCASLICER_PATH=/caminho/para/OrcaSlicer
```

No Docker, isso já aponta para:

```txt
/app/squashfs-root/AppRun
```

### API ocupada

```http
409 Conflict
```

```json
{
  "message": "Slicer is busy"
}
```

Significa que já existe um slicing em execução.

### Profile não encontrado

```http
404 Not Found
```

```json
{
  "message": "Profile not found"
}
```

### Import URL inválida

```json
{
  "message": "URL must be a valid HTTPS URL"
}
```

A API só aceita HTTPS para importação remota.

### Arquivo de profile inválido

```json
{
  "message": "Invalid file type. Only JSON files are allowed"
}
```

ou:

```json
{
  "message": "Invalid JSON profile"
}
```

### Modelo inválido

```json
{
  "message": "Invalid file type. Only STL, STEP and 3MF files are allowed"
}
```

Extensões aceitas:

```txt
.stl
.step
.stp
.3mf
```

## Fluxos recomendados para dashboard

### Fluxo 1: cadastrar profiles manualmente

1. Usuário faz upload de printer profile.
2. Usuário faz upload de preset/process profile.
3. Usuário faz upload de filament profile.
4. Dashboard chama `GET /profiles/{category}` para atualizar listagem.

Exemplo:

```bash
curl -F name=x1c -F file=@printer.json http://localhost:3000/profiles/printers/upload
curl -F name=standard-020 -F file=@preset.json http://localhost:3000/profiles/presets/upload
curl -F name=pla-basic -F file=@filament.json http://localhost:3000/profiles/filaments/upload
```

### Fluxo 2: importar profile por GitHub raw

1. Dashboard envia URL raw JSON.
2. API baixa, valida e salva.
3. API salva `sourceUrl`.
4. Dashboard pode atualizar depois com `update-from-source`.

```bash
curl -X POST http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{"name":"standard-020","url":"https://raw.githubusercontent.com/user/repo/main/preset.json"}'
```

### Fluxo 3: preview antes de fatiar

1. Usuário escolhe printer/preset/filament.
2. Usuário altera opções no dashboard.
3. Dashboard chama `/slice/resolve-profiles`.
4. Dashboard mostra JSON final e warnings.
5. Usuário confirma.
6. Dashboard chama `/slice`.

### Fluxo 4: slicing com override temporário

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=x1c \
  -F preset=standard-020 \
  -F filament=pla-basic \
  -F 'overrides={"preset":{"layer_height":"0.16"}}' \
  -o result.gcode
```

O arquivo `standard-020.json` não é alterado.

### Fluxo 5: monitorar estado do slicer

Antes de permitir novo slicing no dashboard:

```bash
curl http://localhost:3000/slice/status
```

Se status for `processing`, bloqueie o botão de novo slicing.

## Limitações atuais

- Apenas 1 slicing por vez.
- Sem autenticação.
- Sem async job queue.
- Sem banco de dados.
- Importação remota apenas via HTTPS.
- Docker oficial atualmente focado em `linux/amd64`.

## Testes

Rodar:

```bash
go test ./...
go build ./cmd/server
```

O projeto possui testes unitários e integração com slicer fake para validar fluxo sem depender do OrcaSlicer real.
