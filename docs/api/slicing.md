# Slicing

## Endpoint Principal

```
POST /slice
Content-Type: multipart/form-data
```

### Campos

- `file` — modelo (STL, STEP, 3MF)
- `printer` / `preset` / `filament` — nomes de profiles salvos
- `printerProfile` / `presetProfile` / `filamentProfile` — JSONs crus (multipart)
- `resolveProfiles` — resolve `inherits` antes do slicing
- `sanitizeProfiles` — ajusta campos problemáticos para CLI
- `generateImage` — gera thumbnail PNG 160x160
- `enableSupport` — força `enable_support` no preset temporário
- `brimType` — `true` = `auto_brim`, `false` = `no_brim`
- `printSequenceByObject` — `true` = `by object`, `false` = `by layer`
- `overrides` — JSON string com overrides por categoria

### Regras de Prioridade

1. `printerProfile` > `printer`
2. `presetProfile` > `preset`
3. `filamentProfile` > `filament`

### Comportamento de Suportes e Brim

- `enableSupport` e `brimType` são injetados apenas no preset temporário
- Não modificam o profile salvo em disco
- Padrão: valor do profile salvo é mantido

### Sequência de Impressão

- Padrão: `by layer`
- `printSequenceByObject=true` injeta `print_sequence=by object`
