# Resolução e aliases

## `POST /profiles/resolve`

Entrada JSON:

```json
{"category":"presets","name":"meu_processo","overrides":{"layer_height":"0.2"}}
```

Resposta ilustrativa (o conteúdo depende do perfil):

```json
{
  "category": "presets",
  "name": "meu_processo",
  "resolved": {"name":"meu_processo","layer_height":"0.2"},
  "warnings": []
}
```

Não existe campo `sources` na resposta. `warnings` inclui chaves de override inexistentes na base e uso do fallback por tokens.

### Ordem e regras

1. Busca em `DATA_PATH` por nome/arquivo sanitizado e identidades internas, considerando aliases.
2. Busca em `ORCA_PROFILES_PATH`, incluindo pastas de categoria e busca recursiva vendor.
3. Somente se as buscas anteriores falharem, tenta nome com tokens normalizados equivalentes. Rejeita candidatos ambíguos em vez de escolher o primeiro. Gera warning quando usado; não prova equivalência de configurações.
4. Resolve `inherits` string recursivamente; pai ausente retorna 400, ciclo retorna 400, perfil inicial não encontrado retorna 404.
5. Mescla pai e filho (filho prevalece), depois overrides (prevalecem sobre ambos). Objetos têm merge recursivo; arrays/escalares são substituídos.
6. Remove `inherits` do resultado, inclusive se enviado em overrides.

O leitor de arquivos tolera prefixo BOM UTF-8. Arquivos inválidos encontrados durante varredura são ignorados; um candidato direto inválido pode retornar erro. A busca não é um validador de esquema Orca nem resolve ambiguidades semânticas entre perfis parecidos.

## `POST /slice/resolve-profiles`

```bash
curl --fail-with-body http://localhost:3000/slice/resolve-profiles \
  -H 'Content-Type: application/json' \
  -d '{"printer":"minha_impressora","preset":"meu_processo","filament":"meu_filamento","overrides":{"preset":{"layer_height":"0.2"}}}'
```

Retorna objetos opcionais `printer`, `preset`, `filament`, cada um com `category`, `name`, `resolved`, `warnings`. Só processa nomes não vazios. Não executa CLI nem aplica toda a preparação de slicing (como sanitização e controles de suporte/brim/sequência).

## Mudança na 2.5.0

Perfis **nomeados** enviados ao slicing são sempre resolvidos e achatados pela API, mesmo se `resolveProfiles=false`. A flag permanece no contrato por compatibilidade; não é mais um bypass de herança. Warnings são expostos pelos endpoints de resolução, não como resposta adicional no download de slicing: inspecione-os antes.

Perfis enviados diretamente como `printerProfile`, `presetProfile`, `filamentProfile` **não** passam por essa resolução. Forneça JSON completo/sem dependência de `inherits`.

## Aliases persistidos

- `GET /profile-aliases`: array de `{category,from,to}`.
- `POST /profile-aliases`: grava/atualiza um alias e devolve a lista completa (200).
- `DELETE /profile-aliases/{category}/{from}`: remove e devolve lista (200); 404 se ausente. Faça URL-encoding dos componentes.

```json
{"category":"presets","from":"nome_antigo","to":"meu_processo"}
```

O arquivo é `DATA_PATH/profile-aliases.json`. Os três campos devem ser não vazios; use as categorias reconhecidas. Alias é candidato adicional, não renomeia arquivo nem representa encadeamento recursivo garantido. Existem também substituições conhecidas `Neptune4`, `Neptune 4`, `Neptune-4` → `N4` na formação de candidatos.
