# Orca Slicer API - Documentação

API REST em Go para gerenciar profiles JSON e executar slicing com OrcaSlicer CLI.

## Base URL

```txt
http://localhost:3000
```

Em produção, substitua pelo host do container/API.

## Índice

### Conceitos e Operação

- [Visão Geral](api/overview.md)
- [Healthcheck](api/health.md)
- [Configuração](api/configuration.md)
- [Executando](api/running.md)
- [Arquitetura](api/architecture.md)
- [Decisões de Design](api/design-decisions.md)
- [Estrutura de Dados](api/data-structures.md)
- [Erros Comuns](api/errors.md)
- [Fluxos Recomendados](api/flows.md)
- [Checklist Operacional](api/operations.md)
- [Exemplos Práticos](api/examples.md)
- [Checklist para Dashboard](api/dashboard-checklist.md)
- [OpenAPI e Swagger](api/openapi-swagger.md)

---

Consulte também `README.md` na raiz do projeto para visão geral rápida.

### Referência da API

- [Resumo de Endpoints](api/endpoints.md)
- [Profiles](api/profiles.md)
- [Slicing](api/slicing.md)
- [Parâmetros de Slicing](api/slice-parameters.md)
- [Preview](api/preview.md)
- [Compatibilidade Neptune 4](api/neptune4.md)
- [Status e Debug](api/status-debug.md)
- [Resolve e Aliases](api/resolve-aliases.md)

## Links Rápidos

| Recurso | Endpoint |
|---------|----------|
| Health | `GET /health` |
| OpenAPI | `GET /openapi.json` |
| Swagger UI | `GET /api-docs` |
| Slicing | `POST /slice` |
| Preview | `POST /slice/preview` |
| Status | `GET /slice/status` |
| Debug | `GET /slice/debug` |

## Convenções

- JSON usa `Content-Type: application/json`
- Upload/slicing usa `multipart/form-data`
- Erros são retornados em JSON
- Profiles são arquivos `.json` no `DATA_PATH`
- Overrides são temporários e não alteram arquivos salvos
