# Resolve e Aliases

## Resolve de Profile

### `POST /profiles/resolve`

Resolve um único profile aplicando:

1. Overrides enviados
2. `inherits` recursivo
3. Busca em profiles salvos (`DATA_PATH`)
4. Busca em built-ins (`ORCA_PROFILES_PATH`)
5. Aliases configurados

### Body

```json
{
  "category": "presets",
  "name": "0_2mm_standard_0_4_ONP",
  "overrides": {
    "layer_height": "0.2"
  }
}
```

### Resposta

```json
{
  "category": "presets",
  "name": "0_2mm_standard_0_4_ONP",
  "resolved": {
    "type": "process",
    "layer_height": "0.2"
  },
  "sources": []
}
```

### Exemplo

```bash
curl -X POST http://localhost:3000/profiles/resolve \
  -H 'Content-Type: application/json' \
  -d '{"category":"presets","name":"0_2mm_standard_0_4_ONP","overrides":{}}'
```

## Resolve dos Profiles de Slicing

### `POST /slice/resolve-profiles`

Resolve os profiles que seriam usados em um slicing, sem executar o OrcaSlicer.

### Body

```json
{
  "printer": "Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une",
  "preset": "0_2mm_standard_0_4_ONP",
  "filament": "PLA_personalizado1_ONP",
  "overrides": {
    "preset": {
      "enable_support": true
    }
  }
}
```

### Resposta

```json
{
  "printer": { "resolved": {} },
  "preset": { "resolved": {} },
  "filament": { "resolved": {} }
}
```

## Aliases

Aliases ajudam a mapear nomes custom para nomes built-in do OrcaSlicer.

Exemplo:

```txt
Neptune4 -> N4
Neptune 4 -> N4
Neptune-4 -> N4
```

## `GET /profile-aliases`

Lista aliases configurados.

```bash
curl http://localhost:3000/profile-aliases
```

## `POST /profile-aliases`

Cria ou atualiza alias.

### Body

```json
{
  "category": "printers",
  "from": "Neptune4",
  "to": "N4"
}
```

### Exemplo

```bash
curl -X POST http://localhost:3000/profile-aliases \
  -H 'Content-Type: application/json' \
  -d '{"category":"printers","from":"Neptune4","to":"N4"}'
```

## `DELETE /profile-aliases/{category}/{from}`

Remove alias.

```bash
curl -X DELETE http://localhost:3000/profile-aliases/printers/Neptune4
```
