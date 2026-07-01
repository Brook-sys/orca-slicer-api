# Profiles

## Categorias

- `printers`
- `presets`
- `filaments`

## Endpoints

### Listar

```
GET /profiles/{category}
```

Retorna lista com `name`, `size`, `checksum`, `updatedAt`, `sourceUrl`.

### Obter

```
GET /profiles/{category}/{name}
```

Retorna o JSON completo do profile.

### Upload

```
POST /profiles/{category}/upload
Content-Type: multipart/form-data
file=@profile.json
```

### Importar por URL

```
POST /profiles/{category}/import-url
{
  "url": "https://raw.githubusercontent.com/.../profile.json",
  "name": "meu_profile"
}
```

Salva o profile e cria `{name}.source.json` com a URL original.

### Atualizar a partir da source

```
POST /profiles/{category}/{name}/update-from-source
```

Busca a URL original e atualiza o profile salvo.

### Deletar

```
DELETE /profiles/{category}/{name}
```
