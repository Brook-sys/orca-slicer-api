# Exemplos Práticos

## Healthcheck

```bash
curl http://localhost:3000/health
```

## Listar Profiles

```bash
curl http://localhost:3000/profiles/printers
curl http://localhost:3000/profiles/presets
curl http://localhost:3000/profiles/filaments
```

## Upload de Profile

```bash
curl -X POST http://localhost:3000/profiles/presets/upload \
  -F file=@preset.json \
  -F name=meu_preset
```

## Importar Profile por URL

```bash
curl -X POST http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{
    "url": "https://raw.githubusercontent.com/user/repo/main/preset.json",
    "name": "meu_preset"
  }'
```

## Preview para Dashboard

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true
```

## Slice Básico

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

## Slice com Thumbnail

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true \
  -F generateImage=true \
  -o result.gcode
```

## Slice com Suporte, Brim e Thumbnail

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

## Slice sem Suporte e sem Brim

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F enableSupport=false \
  -F brimType=false \
  -o result.gcode
```

## Slice por Objeto

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F printSequenceByObject=true \
  -o result.gcode
```

## Status e Debug

```bash
curl http://localhost:3000/slice/status
curl http://localhost:3000/slice/debug
```
