# Compatibilidade Elegoo Neptune 4

Esta API possui alguns recursos pensados para o fluxo com Elegoo Neptune 4 e OrcaSlicer CLI.

## Thumbnail

Use:

```txt
generateImage=true
```

Ou configure no container:

```txt
GENERATE_IMAGE=true
```

A thumbnail gerada usa:

- PNG 160x160
- Base64 embutido no G-code
- Cor do modelo: `#22948a`
- Fundo transparente
- Formato compatível com o padrão observado em G-code de referência da Neptune 4

## Suportes

Use:

```txt
enableSupport=true
```

ou

```txt
enableSupport=false
```

Isso injeta `enable_support` no preset temporário.

## Brim

Use:

```txt
brimType=true
```

para:

```txt
brim_type=auto_brim
```

Use:

```txt
brimType=false
```

para:

```txt
brim_type=no_brim
```

## Sequência de Impressão

Use:

```txt
printSequenceByObject=true
```

para:

```txt
print_sequence=by object
```

Padrão:

```txt
print_sequence=by layer
```

## Recomendação de Slice

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

## Recomendação de Preview

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true
```
