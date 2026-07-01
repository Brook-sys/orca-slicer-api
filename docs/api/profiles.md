# Profiles

Profiles são arquivos JSON salvos em `DATA_PATH`.

## Categorias

| Categoria | Uso |
|-----------|-----|
| `printers` | Printer / machine profile |
| `presets` | Process / print settings |
| `filaments` | Filament settings |

## `GET /profiles/{category}`

Lista profiles de uma categoria.

### Path Params

| Nome | Tipo | Obrigatório | Descrição |
|------|------|------------:|-----------|
| `category` | string | Sim | `printers`, `presets` ou `filaments` |

### Resposta

```json
[
  {
    "name": "0_2mm_standard_0_4_ONP",
    "size": 12345,
    "checksum": "sha256...",
    "updatedAt": "2026-07-01T00:00:00Z",
    "sourceUrl": "https://raw.githubusercontent.com/..."
  }
]
```

### Exemplo

```bash
curl http://localhost:3000/profiles/presets
```

## `GET /profiles/{category}/{name}`

Retorna um profile JSON salvo.

### Exemplo

```bash
curl http://localhost:3000/profiles/presets/0_2mm_standard_0_4_ONP
```

## `POST /profiles/{category}/upload`

Faz upload de um profile JSON.

### Form Fields

| Campo | Tipo | Obrigatório | Descrição |
|-------|------|------------:|-----------|
| `file` | file JSON | Sim | Profile JSON |
| `name` | string | Não | Nome salvo. Se omitido, usa o nome do arquivo |

### Exemplo

```bash
curl -X POST http://localhost:3000/profiles/presets/upload \
  -F file=@preset.json \
  -F name=meu_preset
```

## `POST /profiles/{category}/import-url`

Importa profile a partir de URL HTTPS raw.

### Body

```json
{
  "url": "https://raw.githubusercontent.com/user/repo/main/profile.json",
  "name": "meu_profile"
}
```

### Regras

- URL deve ser HTTPS
- Conteúdo deve ser JSON válido
- Cria `{name}.source.json` para update posterior

### Exemplo

```bash
curl -X POST http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{"url":"https://raw.githubusercontent.com/.../preset.json","name":"meu_preset"}'
```

## `POST /profiles/{category}/{name}/update-from-source`

Atualiza um profile importado usando a URL salva em `{name}.source.json`.

### Exemplo

```bash
curl -X POST http://localhost:3000/profiles/presets/meu_preset/update-from-source
```

## `DELETE /profiles/{category}/{name}`

Remove profile e source tracking associado.

### Exemplo

```bash
curl -X DELETE http://localhost:3000/profiles/presets/meu_preset
```

## Erros Comuns

| Status | Causa |
|--------|-------|
| `400` | categoria inválida, JSON inválido, URL inválida |
| `404` | profile não encontrado |
| `500` | erro de filesystem ou download |
