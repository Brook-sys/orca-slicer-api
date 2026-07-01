# Fluxos Recomendados para Dashboard

## Fluxo 1 — Preview antes do Slice Final

Use quando o dashboard precisa mostrar thumbnail e avisar sobre suporte.

### 1. Gerar Preview

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true
```

Resposta:

```json
{
  "usesSupport": true,
  "printTime": 12450.3,
  "filamentUsedG": 18.7,
  "filamentUsedMm": 12450.3,
  "thumbnail": "iVBORw0KGgo..."
}
```

### 2. Decidir no Dashboard

Se `usesSupport=true`:

- Mostrar aviso: "Este modelo gera suportes com o profile atual"
- Permitir continuar com suportes ativados
- Permitir desativar suportes manualmente

### 3. Executar Slice Final

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true \
  -F enableSupport=true \
  -F brimType=true \
  -F generateImage=true \
  -o result.gcode
```

## Fluxo 2 — Slice Direto com Controles

Use quando o usuário já escolheu as opções.

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F enableSupport=false \
  -F brimType=false \
  -F printSequenceByObject=false \
  -F generateImage=true \
  -o result.gcode
```

## Fluxo 3 — Profiles Multipart sem Salvar

Use quando o dashboard fornece profiles temporários.

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printerProfile=@printer.json \
  -F presetProfile=@preset.json \
  -F filamentProfile=@filament.json \
  -F enableSupport=false \
  -F generateImage=true \
  -o result.gcode
```

## Fluxo 4 — Diagnóstico de Preset

Use para verificar resolução de `inherits` sem fatiar.

```bash
curl -X POST http://localhost:3000/profiles/resolve \
  -H 'Content-Type: application/json' \
  -d '{
    "category": "presets",
    "name": "0_2mm_standard_0_4_ONP",
    "overrides": {}
  }'
```

## Fluxo 5 — Debug após Erro

```bash
curl http://localhost:3000/slice/debug
```

Verifique:

- `command`
- `args`
- `preset`
- `printer`
- `filament`
- `output`
- `resultJSON.error_string`
