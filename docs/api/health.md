# Health e métricas

## `GET /health`

Retorna 200 com `status:healthy`, ou 503 com `status:unhealthy` se falhar um check essencial: existência/escrita de `DATA_PATH`, configuração/existência/permissão executável de `ORCASLICER_PATH`.

Campos: `status`, `dataPath`, `dataPathExists`, `dataPathWritable`, `orcaSlicerPath`, `orcaSlicerConfigured`, `orcaSlicerExists`, `orcaSlicerExecutable`, `orcaProfilesPath`, `orcaProfilesPathExists`, `orcaVersion`, `generateImage`, `resolveProfiles`, `sanitizeProfiles`.

```bash
curl --fail-with-body http://localhost:3000/health
```

## O que não comprova

- `orcaVersion` vem de `ORCA_VERSION`, não da leitura do executável. Confira também o banner de `AppRun --help`.
- `orcaProfilesPathExists` é informativo: **não participa da decisão healthy/unhealthy**. Health 200 pode coexistir com raiz JSON errada ou pais indisponíveis.
- Não executa CLI, resolve perfis ou valida slice/impressão. Uma permissão de execução não garante bibliotecas ou arquitetura corretas.
- O teste de escrita cria/remove um arquivo temporário no diretório de dados.

Na imagem 2.5.0, a raiz JSON é `/app/profiles-json/resources/profiles`; o AppImage original mantém os próprios recursos em separado.

## `GET /metrics`

Retorna **JSON**, não Prometheus: `uptime_seconds`, `goroutines`, `memory_alloc_mb`, estatísticas `jobs` e indicador `cache_enabled` quando configurados. Memória é do runtime Go, não consumo total do CLI. Não há contadores de hit/miss nem percentis de latência nesse endpoint. Veja [operação](operations.md).
