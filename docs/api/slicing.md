# Slicing

## `POST /slice`

Executa slicing síncrono usando OrcaSlicer CLI e retorna o arquivo gerado.

- Entrada: `multipart/form-data`
- Saída: G-code, 3MF ou ZIP
- Concorrência: apenas 1 slicing por vez
- Overrides são temporários
- Profiles salvos não são modificados

## Multipart Fields

### Arquivo

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `file` | file | Sim | Modelo de entrada. Aceita STL, STEP e 3MF |

### Profiles por Nome

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `printer` | string | Condicional | Nome do profile em `printers` |
| `preset` | string | Condicional | Nome do profile em `presets` |
| `filament` | string | Condicional | Nome do profile em `filaments` |

### Profiles Multipart

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `printerProfile` | file JSON | Não | Profile de printer enviado diretamente |
| `presetProfile` | file JSON | Não | Profile de preset enviado diretamente |
| `filamentProfile` | file JSON | Não | Um ou mais profiles de filamento |

Prioridade:

1. `printerProfile` substitui `printer`
2. `presetProfile` substitui `preset`
3. `filamentProfile` substitui `filament`

### Flags de Slicing

| Campo | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `exportType` | string | `gcode` | `gcode` ou `3mf` |
| `plate` | string | `1` | Placa a fatiar |
| `arrange` | bool | `false` | Executa arrange automático |
| `orient` | bool | `false` | Executa orient automático |
| `multicolorOnePlate` | bool | `false` | Passa `--allow-multicolor-oneplate` |
| `resolveProfiles` | bool | `false` | Resolve `inherits` e built-ins antes do slicing |
| `sanitizeProfiles` | bool | `false` | Ajusta profiles temporários para compatibilidade com Orca CLI |
| `generateImage` | bool | `false` ou `GENERATE_IMAGE` | Gera thumbnail PNG 160x160 no G-code |

### Overrides de Preset Temporário

| Campo | Tipo | Padrão | Efeito |
|-------|------|--------|--------|
| `enableSupport` | bool opcional | valor do profile | Injeta `enable_support=true/false` |
| `brimType` | bool opcional | valor do profile | `true` → `auto_brim`; `false` → `no_brim` |
| `printSequenceByObject` | bool | `false` | `true` → `by object`; `false` → `by layer` |
| `overrides` | JSON string | `{}` | Overrides por `printer`, `preset`, `filament` |

Observações:

- `enableSupport` e `brimType` só são aplicados se enviados.
- `printSequenceByObject` sempre define `print_sequence` no preset temporário.
- Nenhum desses campos altera profiles salvos em disco.

## Exemplo Básico

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true \
  -o result.gcode
```

## Exemplo com Thumbnail, Suporte e Brim

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true \
  -F generateImage=true \
  -F enableSupport=true \
  -F brimType=true \
  -F printSequenceByObject=false \
  -o result.gcode
```

## Resposta de Sucesso

Retorna arquivo binário:

- `Content-Type: text/plain` para G-code
- `Content-Type: application/zip` para múltiplos arquivos
- `Content-Type: model/3mf` ou similar para 3MF

Headers úteis podem incluir metadata extraída do G-code/3MF.

## Erros Comuns

| Status | Causa |
|--------|-------|
| `400` | arquivo ausente, tipo inválido, overrides inválidos |
| `409` | slicer ocupado |
| `500` | OrcaSlicer falhou |
| `408` | slicing cancelado ou timeout |

Use `GET /slice/debug` para investigar falhas.
