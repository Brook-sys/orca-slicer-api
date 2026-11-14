# OpenAPI e Swagger UI

- `GET /openapi.json`: especificação OpenAPI 3.0.3, mantida manualmente em `internal/docs/docs.go`.
- `GET /api-docs`: HTML da Swagger UI. CSS/JavaScript vêm de CDN externa (`unpkg.com`); uso offline/rede restrita pode não funcionar.

```bash
curl --fail-with-body http://localhost:3000/openapi.json -o openapi.json
```

A versão `info.version` da especificação não identifica necessariamente o OrcaSlicer. O esquema não é gerado dos handlers nem garante cobertura atualizada de todas as rotas.

## Divergências conhecidas

A especificação ainda contém exemplos/schema de erro com `error`, mas os handlers retornam **`message`**. Também descreve `resolveProfiles` como opt-in, enquanto o código atual sempre resolve perfis nomeados. O inventário completo e os contratos atuais estão em [rotas](endpoints.md), [parâmetros](slice-parameters.md) e [erros](errors.md).

Esta revisão é somente documental: não modifica o Go que contém a especificação. Não gere um cliente assumindo fidelidade completa sem comparar esquema e handlers. Teste multipart e download por HTTP real; o Swagger não substitui integração com o CLI.

A UI não adiciona autenticação: qualquer cliente com acesso à API pode executar operações reais. Não expor a interface como mecanismo de autorização.
