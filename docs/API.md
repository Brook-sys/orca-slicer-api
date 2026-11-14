# Documentação da API

API HTTP em Go para perfis JSON e OrcaSlicer CLI. Base dos exemplos: `http://localhost:3000`. Não há autenticação. Use rede confiável ou proxy com controle de acesso.

**Baseline:** atualização para OrcaSlicer 2.5.0 enviada para build, imagem somente `linux/amd64`. Cubo e STL de 5.952 triângulos foram fatiados com AppImage original e perfis achatados pela API; integração HTTP completa e imagem final ainda pendentes. Não houve equivalência com a referência 2.4.0 ou impressão física validada. Não representa deploy concluído. Veja [evidências](api/overview.md).

## Começar

1. [Guia de primeiros passos](GETTING_STARTED.md) e [resumo no README](../README.md)
2. [Execução](api/running.md) e [configuração](api/configuration.md)
3. [Perfis](api/profiles.md) e [resolução/aliases](api/resolve-aliases.md)
4. [Exemplos](api/examples.md), [slicing/jobs](api/slicing.md) e [parâmetros](api/slice-parameters.md)
5. [Operação, segurança e limitações](api/operations.md)

## Referência

- [Todas as rotas](api/endpoints.md)
- [Health](api/health.md)
- [Preview](api/preview.md)
- [Status/debug](api/status-debug.md)
- [Contratos de dados](api/data-structures.md)
- [Erros](api/errors.md)
- [OpenAPI/Swagger](api/openapi-swagger.md)
- [Neptune 4 e thumbnails](api/neptune4.md)

## Contexto e integração

- [Visão geral](api/overview.md)
- [Arquitetura](api/architecture.md)
- [Decisões de design](api/design-decisions.md)
- [Fluxos de integração](api/flows.md)
- [Checklist de dashboard](api/dashboard-checklist.md)
- [Roadmap: propostas, não funcionalidades atuais](ROADMAP.md)

## Convenções

- JSON em `application/json`; modelos/perfis enviados em `multipart/form-data`.
- Erros dos handlers: `{"message":"..."}`. Erros de roteamento do `net/http` podem ser texto.
- Overrides e sanitização de slicing são temporários; upload/import alteram disco.
- Perfis nomeados são sempre achatados pela API, independentemente de `resolveProfiles`.
- Campos retornados por health/debug podem revelar caminhos e conteúdo dos perfis: não exponha esses endpoints publicamente.
- Exemplos usam arquivos e nomes fictícios que você deve fornecer, não dados de teste reais do operador.
