# Perfis JSON

## Armazenamento e validação

Categorias: `printers` (machine), `presets` (process), `filaments` (filament). Perfis de usuário ficam em `DATA_PATH/{category}/{name}.json`.

- `name` é obrigatório, com expressão `^[a-zA-Z0-9_-]+$`; não é inferido do arquivo.
- Máximo de **4.000.000 bytes** por perfil salvo/importado.
- A validação aceita JSON sintaticamente válido; não valida esquema Orca, compatibilidade física ou comandos de G-code.
- Upload/import não toleram BOM UTF-8. O leitor de resolução em disco o remove, mas isso não muda a validação de entrada.

## Operações

| Método e rota | Entrada | Resposta |
|---|---|---|
| `GET /profiles/{category}` | Categoria | Array com `name`, `size`, `checksum` SHA-256, `updatedAt` RFC3339 e `sourceUrl` opcional |
| `GET /profiles/{category}/{name}` | Nome salvo | Conteúdo JSON original |
| `POST /profiles/{category}/upload` | Multipart: `file` terminado em `.json`, `name` obrigatório | 201, `{name,checksum,updated}`; upload retorna `updated:false` mesmo tendo gravado |
| `POST /profiles/{category}/import-url` | JSON `{name,url,overwrite}` | 201, `{name,checksum,sourceUrl,updated}` |
| `POST /profiles/{category}/{name}/update-from-source` | Sem corpo necessário | 200, mesma estrutura de import; `updated:false` se checksum da fonte não mudou |
| `DELETE /profiles/{category}/{name}` | Categoria/nome | 204; remove também `.source.json` |

Upload substitui um perfil existente. Import exige `overwrite:true` para substituir; caso contrário retorna 409. Import grava `{name}.source.json` com URL/checksum/data. Update exige source salva (400 se ausente), baixa novamente e compara o checksum com o registro da fonte.

```bash
curl --fail-with-body http://localhost:3000/profiles/presets/upload \
  -F file=@preset.json -F name=meu_processo

curl --fail-with-body http://localhost:3000/profiles/presets/import-url \
  -H 'Content-Type: application/json' \
  -d '{"name":"meu_processo","url":"https://example.org/preset.json","overwrite":true}'
```

A URL é ilustrativa: forneça um endpoint que realmente devolva JSON raw. O cliente tem timeout de 20 segundos e aceita URL inicial HTTPS; não há allowlist de destinos nem proteção SSRF completa. Restrinja acesso e egress.

## Built-ins: resolução não é listagem

`ORCA_PROFILES_PATH` aponta para a raiz dos JSON de fabricante. Para 2.5.0 na imagem: `/app/profiles-json/resources/profiles`, obtida do repositório upstream na mesma tag; os `.opc` originais continuam no AppImage.

`GET /profiles/builtins/{category}` procura somente `{raiz}/{category}` ou `{raiz}/{machine|process|filament}`, listando apenas JSON diretamente nessa pasta. Retorna itens `{name,path}`. **Não percorre fabricantes/subpastas**: no layout vendor normal do contêiner JSON (`{raiz}/Elegoo/machine`, por exemplo), retorna 404 mesmo com resolução por nome funcional. Em layouts rasos alternativos, a lista ainda pode ser parcial. Não use essa rota como catálogo completo nem como teste definitivo de disponibilidade de perfis.

A resolução por nome faz busca recursiva e tem regras próprias: veja [resolve e aliases](resolve-aliases.md).
