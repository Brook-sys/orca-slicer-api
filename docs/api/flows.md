# Fluxos Recomendados para Dashboard

## 1. Preview Rápido (detectar suporte)

```
POST /slice/preview
→ retorna usesSupport + thumbnail + tempos
```

Se `usesSupport=true`:
- Mostrar aviso ao usuário
- Oferecer opção de ajustar preset ou continuar

## 2. Slicing com Controle Total

```
POST /slice
  -F enableSupport=true
  -F brimType=true
  -F printSequenceByObject=false
  -F generateImage=true
  -F resolveProfiles=true
  -F sanitizeProfiles=true
```

## 3. Slicing com Profiles Multipart (sem salvar)

```
POST /slice
  -F file=@model.stl
  -F printerProfile=@printer.json
  -F presetProfile=@preset.json
  -F filamentProfile=@filament1.json
  -F filamentProfile=@filament2.json
  -F enableSupport=false
```

## 4. Diagnóstico de Preset

```
POST /profiles/resolve
{
  "category": "presets",
  "name": "meu_preset",
  "overrides": { ... }
}
```

Útil para verificar se `inherits` está sendo resolvido corretamente.
