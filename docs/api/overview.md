# Visão Geral

Esta documentação cobre a API REST em Go para gerenciar profiles JSON e executar slicing com OrcaSlicer CLI.

## Fluxo Principal

```
Dashboard / Cliente
  -> envia modelo + profiles (por nome ou JSON cru no multipart) + overrides
  -> API cria workdir temporário
  -> API grava profiles temporários
  -> API executa OrcaSlicer CLI
  -> API coleta G-code/3MF
  -> API extrai metadata
  -> API retorna arquivo ou JSON (preview)
```

## Modos de Profiles

1. **Por nome salvo**: `printer`/`preset`/`filament` apontam para `DATA_PATH/{category}/{name}.json`
2. **Por arquivo multipart**: `printerProfile`/`presetProfile`/`filamentProfile` enviam JSONs crus

## Modos de Resolução

- Padrão: usa JSON salvo cru + overrides temporários
- `resolveProfiles=true`: resolve `inherits` e built-ins antes do slicing
- Endpoints de diagnóstico: `/profiles/resolve` e `/slice/resolve-profiles`

## O Que Está Implementado

- Healthcheck completo
- CORS configurável
- Logs HTTP estruturados
- CRUD de profiles JSON
- Importação por URL HTTPS raw
- Source tracking de profiles importados
- Update manual a partir da source URL
- Listagem rica com metadata (name, size, checksum, updatedAt, sourceUrl)
- Preview de profile resolvido
- Slicing síncrono com lock global (1 por vez)
- Timeout e cancelamento
- Status persistido do último slicing
- Metadata de G-code/3MF
- Retorno de arquivo ou ZIP
- OpenAPI + Swagger UI
- Docker com OrcaSlicer oficial amd64
- GHCR via GitHub Actions

## O Que Não Está Implementado (Por Design)

- Slicing assíncrono
- Autenticação / API key
- Banco de dados
- Fila de jobs simultâneos
- Execução paralela de slicing
