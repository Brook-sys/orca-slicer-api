# Erros

## Formato de Erro

A API retorna erros em JSON:

```json
{
  "error": "Mensagem do erro"
}
```

## Códigos Comuns

| Código | Mensagem comum | Causa |
|--------|----------------|-------|
| `400` | `Invalid multipart form` | Form multipart inválido |
| `400` | `Model file is required` | Campo `file` ausente |
| `400` | `Invalid file type` | Arquivo não é STL/STEP/3MF |
| `400` | `Model file is too large` | Arquivo maior que 100MB |
| `400` | `Invalid overrides JSON` | Campo `overrides` inválido |
| `409` | `Slicer is busy` | Já existe slicing em andamento |
| `408` | `Slicing cancelled or timed out` | Timeout ou cancelamento |
| `500` | `Slicing failed: ...` | OrcaSlicer retornou erro |
| `500` | `Image generation failed: ...` | Falha ao gerar thumbnail |

## Erros Específicos de Profiles

| Mensagem | Causa | Correção |
|----------|-------|----------|
| `inherits from X, but parent profile was not found` | Preset depende de built-in/pai não resolvido | Use `resolveProfiles=true` |
| `The input preset file is invalid and can not be parsed` | Profile custom com campo incompatível | Use `sanitizeProfiles=true` |
| `The selected printer is not compatible with the process preset` | Compatibilidade printer/preset | Use profiles resolvidos e sanitizados |
| `Invalid parameter value(s) included in the 3mf file` | Campo inválido no profile/modelo | Consulte `/slice/debug` |

## Debug Recomendado

1. Chame `/slice/debug`
2. Verifique `resultJSON.error_string`
3. Verifique `preset`, `printer`, `filament`
4. Refaça a chamada com:
   - `resolveProfiles=true`
   - `sanitizeProfiles=true`
5. Se necessário, use `/profiles/resolve` para ver o profile final

## Exemplo

```bash
curl http://localhost:3000/slice/debug
```
