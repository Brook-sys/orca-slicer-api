# OpenAPI e Swagger

A API expõe documentação interativa via OpenAPI 3.0.

## Endpoints

| Endpoint | Descrição |
|----------|-----------|
| `GET /openapi.json` | Especificação OpenAPI em JSON |
| `GET /api-docs` | Swagger UI |

## Usando Swagger UI

Abra no navegador:

```txt
http://localhost:3000/api-docs
```

A interface permite:

- Ver todos os endpoints
- Ver schemas de request/response
- Testar chamadas simples
- Consultar exemplos de erro

## Baixar OpenAPI JSON

```bash
curl http://localhost:3000/openapi.json -o openapi.json
```

## Gerar Cliente a partir do OpenAPI

Exemplo com `openapi-generator`:

```bash
openapi-generator-cli generate \
  -i http://localhost:3000/openapi.json \
  -g typescript-fetch \
  -o ./orca-api-client
```

## Tags no Swagger

A documentação é agrupada por tags:

| Tag | Descrição |
|-----|-----------|
| `Health` | Healthcheck |
| `Profiles` | CRUD/importação de profiles |
| `Aliases` | Alias de profiles |
| `Resolve` | Resolução de inherits/built-ins |
| `Slicing` | Slice, preview, status e debug |
| `Docs` | OpenAPI e Swagger |

## Observações

- Endpoints multipart podem exigir ajuste manual no Swagger UI dependendo do navegador.
- Para testes completos de `/slice` e `/slice/preview`, `curl` costuma ser mais confiável.
- O OpenAPI deve ser mantido alinhado com `docs/api/*.md`.
