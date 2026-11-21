# OpenAPI e Swagger UI

- `GET /openapi.json`: especificação OpenAPI 3.0.3, mantida manualmente em `internal/docs/docs.go`.
- `GET /api-docs`: HTML da Swagger UI. CSS/JavaScript vêm de CDN externa (`unpkg.com`); uso offline/rede restrita pode não funcionar.

```bash
curl --fail-with-body http://localhost:3000/openapi.json -o openapi.json
```

A versão `info.version` da especificação não identifica necessariamente o OrcaSlicer. O esquema não é gerado dos handlers nem garante cobertura de todas as respostas possíveis do `net/http`.

## Contrato de slicing representado

A especificação descreve um único STL, quantidade/mesa estritas, somente G-code, erros com **`message`**, resposta 422, headers sync e o esquema completo de toolpaths/limits do preview. Perfis nomeados são documentados como sempre resolvidos; `resolveProfiles` permanece uma flag legada.

A especificação continua manual: ao alterar handlers, revise-a junto com [rotas](endpoints.md), [parâmetros](slice-parameters.md), [preview](preview.md) e [erros](errors.md). Teste multipart e download por HTTP real; o Swagger não substitui integração com o CLI.

A UI não adiciona autenticação: qualquer cliente com acesso à API pode executar operações reais. Não expor a interface como mecanismo de autorização.
