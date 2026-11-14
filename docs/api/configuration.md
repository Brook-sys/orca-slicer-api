# Configuração

Valores lidos por `internal/config/config.go` na inicialização:

| Variável | Default do binário Go | Uso |
|---|---|---|
| `PORT` | `3000` | Escuta em todas as interfaces (`:<porta>`) |
| `DATA_PATH` | `data` | Perfis, aliases, estado, jobs e cache |
| `ORCASLICER_PATH` | vazio | Executável; necessário para slicing saudável |
| `ORCA_PROFILES_PATH` | `/app/squashfs-root/resources/profiles` | Raiz JSON usada para resolver pais |
| `ORCA_VERSION` | vazio | Valor informativo de `/health` |
| `SLICE_TIMEOUT_SECONDS` | `1800` | Prazo do processo CLI; aceita inteiro positivo |
| `GENERATE_IMAGE` | `false` | Default de thumbnail |
| `RESOLVE_PROFILES` | `false` | Flag legada: não desativa resolução de perfis nomeados |
| `SANITIZE_PROFILES` | `false` | Default de sanitização |
| `CORS_ORIGINS` | `*` | Origins separados por vírgula |

Booleans reconhecem exatamente `true`, `1` ou `on`; outros valores são falsos. Timeout inválido/não positivo usa 1800.

## Docker e 2.5.0

O Dockerfile define `DATA_PATH=/app/data`, `ORCASLICER_PATH=/app/squashfs-root/AppRun`, `ORCA_VERSION` pelo build arg e **`ORCA_PROFILES_PATH=/app/profiles-json/resources/profiles`**. O default Go antigo não serve como fonte JSON completa para 2.5.0: configure a raiz explicitamente em execução fora da imagem.

O stage `vendorprofiles` preserva `resources/profiles` ao copiar os JSON upstream da mesma tag do AppImage. `/app/profiles-json` sozinho é o nível errado. A existência da raiz não prova que um pai específico está disponível.

## Precedência e segurança

`generateImage`, `sanitizeProfiles` e a flag legada `resolveProfiles` usam **valor do formulário OU default de ambiente**. Enviar `false` não desativa um default global `true`.

CORS não autentica nem bloqueia clientes HTTP. O header `Authorization` é permitido pelo middleware, mas não é validado. Restrinja a exposição por rede/proxy; veja [operação](operations.md).
