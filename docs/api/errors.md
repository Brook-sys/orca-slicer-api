# Erros Comuns

| Código | Mensagem | Causa Comum |
|--------|----------|-------------|
| 400 | Model file is required | Campo `file` ausente |
| 400 | Invalid file type | Não é STL/STEP/3MF |
| 400 | Model file is too large | > 100MB |
| 409 | Slicer is busy | Já existe um slicing em andamento |
| 500 | Slicing failed | Erro do OrcaSlicer (ver `/slice/debug`) |
| 500 | The input preset file is invalid | Preset com `from=User` ou campos inválidos |
| 500 | inherits from X, but parent profile was not found | `resolveProfiles=false` com preset que herda |

## Dicas de Depuração

1. Use `/slice/debug` para ver comando, perfis temporários, stdout/stderr
2. Verifique se `resolveProfiles=true` quando o preset usa `inherits`
3. Verifique se `sanitizeProfiles=true` quando preset tem `from=User` ou `small_perimeter_speed`
4. Para Neptune 4, use `enableSupport` e `brimType` se o preset salvo não tiver esses campos
