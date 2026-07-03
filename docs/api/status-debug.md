# Status e Debug

## `GET /slice/status`

Retorna o status persistido do último slicing.

### Resposta

```json
{
  "status": "completed",
  "startedAt": "2026-07-01T00:00:00Z",
  "finishedAt": "2026-07-01T00:02:00Z",
  "errorMessage": "",
  "files": ["/tmp/slice/output/result.gcode"],
  "metadata": {
    "printTime": 12000,
    "filamentUsedG": 18.5,
    "filamentUsedMm": 6200
  }
}
```

### Status Possíveis

| Status | Descrição |
|--------|-----------|
| `idle` | Nenhum job ativo conhecido |
| `processing` | Slicing em execução |
| `completed` | Último slicing concluído |
| `failed` | Último slicing falhou |
| `cancelled` | Último slicing cancelado/timeout |

### Exemplo

```bash
curl http://localhost:3000/slice/status
```

## `GET /slice/debug`

Retorna informações detalhadas do último slicing.

Use este endpoint para diagnosticar erros do OrcaSlicer CLI.

### Resposta

```json
{
  "startedAt": "2026-07-01T00:00:00Z",
  "finishedAt": "2026-07-01T00:02:00Z",
  "workdir": "/tmp/slice-123",
  "inputPath": "/tmp/slice-123/input/model.stl",
  "outputDir": "/tmp/slice-123/output",
  "command": "/app/squashfs-root/AppRun",
  "args": ["--slice", "1", "--outputdir", "/tmp/..."],
  "settings": {},
  "printer": {},
  "preset": {},
  "filament": {},
  "filaments": [],
  "output": "stdout/stderr do OrcaSlicer",
  "resultJSON": {},
  "slicerError": "",
  "errorMessage": ""
}
```

### Exemplo

```bash
curl http://localhost:3000/slice/debug
```

## Logs de Slicing

No início de cada slicing, a API registra flags principais em log estruturado:

```txt
slicing started file=model.stl export_type=gcode generate_image=true enable_support=true brim_type=auto_brim print_sequence_by_object=false resolve_profiles=true sanitize_profiles=true arrange=false orient=false
```

Campos logados:

| Campo | Descrição |
|-------|-----------|
| `file` | Nome base do arquivo enviado |
| `export_type` | `gcode` ou `3mf` |
| `generate_image` | Se thumbnail será gerada |
| `enable_support` | `true`, `false` ou `profile_default` |
| `brim_type` | `auto_brim`, `no_brim` ou `profile_default` |
| `print_sequence_by_object` | Sequência por objeto |
| `resolve_profiles` | Resolução de profiles ativa |
| `sanitize_profiles` | Sanitização ativa |
| `arrange` | Arrange automático |
| `orient` | Orient automático |

A API não loga o conteúdo dos arquivos enviados nem JSON completo de profiles no log padrão.

## Recomendações de Debug

1. Verifique `command` e `args` para reproduzir localmente.
2. Verifique `preset`, `printer` e `filament` para confirmar overrides temporários.
3. Verifique `resultJSON.error_string` quando o Orca retorna erro estruturado.
4. Se profiles custom falharem, teste com `resolveProfiles=true` e `sanitizeProfiles=true`.
