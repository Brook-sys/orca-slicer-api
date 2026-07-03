# Parâmetros de Slicing

Esta página centraliza os parâmetros multipart usados por `/slice` e `/slice/preview`.

## Arquivo de Entrada

| Campo | Tipo | Obrigatório | Usado em | Descrição |
|-------|------|------------:|----------|-----------|
| `file` | file | Sim | `/slice`, `/slice/preview` | Modelo STL, STEP ou 3MF |

## Profiles por Nome

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `printer` | string | Condicional | Nome do profile salvo em `printers` |
| `preset` | string | Condicional | Nome do profile salvo em `presets` |
| `filament` | string | Condicional | Nome do profile salvo em `filaments` |

## Profiles Multipart

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `printerProfile` | file JSON | Não | Profile de printer enviado diretamente |
| `presetProfile` | file JSON | Não | Profile de preset/process enviado diretamente |
| `filamentProfile` | file JSON | Não | Um ou mais profiles de filamento |

Prioridade:

```txt
printerProfile > printer
presetProfile > preset
filamentProfile > filament
```

## Resolução e Sanitização

| Campo | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `resolveProfiles` | bool | `false` ou env `RESOLVE_PROFILES` | Resolve `inherits` e built-ins antes do slicing |
| `sanitizeProfiles` | bool | `false` ou env `SANITIZE_PROFILES` | Ajusta fields temporários conhecidos por quebrar o Orca CLI |

## Controle de Output

| Campo | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `exportType` | string | `gcode` | `gcode` ou `3mf` |
| `plate` | string | `1` | Placa a ser fatiada |
| `arrange` | bool | `false` | Executa arrange automático |
| `orient` | bool | `false` | Executa orient automático |
| `multicolorOnePlate` | bool | `false` | Passa `--allow-multicolor-oneplate` |

## Controles Temporários de Preset

| Campo | Tipo | Padrão | Efeito |
|-------|------|--------|--------|
| `enableSupport` | bool opcional | valor do preset | Injeta `enable_support=true/false` |
| `brimType` | bool opcional | valor do preset | `true` → `auto_brim`; `false` → `no_brim` |
| `printSequenceByObject` | bool | `false` | `true` → `print_sequence=by object`; `false` → `by layer` |

## Thumbnail

| Campo | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `generateImage` | bool | `false` ou env `GENERATE_IMAGE` | Injeta thumbnail PNG 160x160 no G-code |

## Overrides JSON

| Campo | Tipo | Padrão | Descrição |
|-------|------|--------|-----------|
| `overrides` | string JSON | `{}` | Overrides por `printer`, `preset`, `filament` |

Exemplo:

```json
{
  "preset": {
    "layer_height": "0.2",
    "enable_support": true
  },
  "filament": {
    "filament_flow_ratio": ["1"]
  }
}
```

Como multipart:

```bash
-F 'overrides={"preset":{"layer_height":"0.2"}}'
```

## Diferença entre `/slice` e `/slice/preview`

| Campo | `/slice` | `/slice/preview` |
|-------|----------|------------------|
| `enableSupport` | Respeitado se enviado | Ignorado; preview força `true` |
| `generateImage` | Respeitado/env | Ignorado; preview sempre gera |
| `brimType` | Respeitado | Respeitado |
| `printSequenceByObject` | Respeitado | Respeitado |
| Retorno | Arquivo G-code/3MF/ZIP | JSON |
