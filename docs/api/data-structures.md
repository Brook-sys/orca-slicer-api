# Contratos de dados

Referências: `internal/slicer/types.go`, `overrides.go`, `state.go`, `internal/profiles/store.go`, `internal/jobs/jobs.go`. Não confundir structs internas com respostas HTTP.

## Metadados

```json
{"printTime":412,"filamentUsedG":1.0,"filamentUsedMm":340.83}
```

`printTime` em segundos; massa em gramas; comprimento em milímetros. Valores ilustrativos, não medições físicas. Sync retorna estimativas em headers; preview e estado retornam JSON.

## Perfil resolvido

```json
{"category":"presets","name":"meu_processo","resolved":{},"warnings":[]}
```

`resolved` contém o objeto mesclado sem `inherits`. `warnings` é array de strings, não um grafo de fontes. Resolução de conjunto retorna chaves opcionais `printer`, `preset`, `filament` com essa estrutura.

## Perfil salvo/importado

Listagem: `{name,size,checksum,updatedAt,sourceUrl?}`. `size` em bytes, checksum SHA-256 hexadecimal. Upload/import/update: `{name,checksum,sourceUrl?,updated}`. Leia as diferenças de overwrite/updated em [perfis](profiles.md).

## Job

Campos HTTP: `id`, `status`, `progress` (0–100), `createdAt`, `updatedAt`, `error` opcional. Estados: pending/slicing/completed/failed. `ResultFiles`, `Workdir` e função de cancelamento não são serializados; não fazem parte do JSON de job nem são recuperados para download após restart.

## Settings e resultado interno

Settings de slicing são parseadas do multipart; perfis diretos são arquivos, não campos JSON embutidos no body. Nomes/flags/overrides estão em [parâmetros](slice-parameters.md). `/slice/resolve-profiles` usa JSON com nomes/overrides, não upload de perfis.

`Result{Files,Workdir,Metadata}` é interno: `/slice` retorna arquivo/ZIP, não esse objeto. `PreviewResult` também é interno; o handler expõe `usesSupport`, `printTime`, `filamentUsedG`, `filamentUsedMm`, `thumbnail`.

Estado e debug globais têm contratos distintos dos jobs: veja [status/debug](status-debug.md).
