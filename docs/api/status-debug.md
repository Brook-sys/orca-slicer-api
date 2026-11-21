# Estado global e debug

## `GET /slice/status`

Estado persistido do último processamento global, não lista de jobs. Campos: `status`, `startedAt`, `finishedAt`, `errorMessage`, `files`, `metadata` (alguns omitidos quando vazios). Estados: `idle`, `processing`, `completed`, `failed`, `cancelled`.

`metadata` usa `printTime` (segundos), `filamentUsedG`, `filamentUsedMm`. Os caminhos `files` são internos e podem já ter sido removidos após uma resposta sync. Não são URLs de download.

## `GET /slice/debug`

Diagnóstico global persistido em `DATA_PATH/slice-debug.json`. Campos possíveis:

- `startedAt`, `finishedAt`, `workdir`, `inputPath`, `outputDir`.
- `command`, `args`, `settings`.
- `printer`, `preset`, `filament`, `filaments`: perfis temporários efetivos.
- `output`: stdout/stderr combinado, retido até 1 MiB com marcador de truncamento; `slicerError`: erro do resultado do Orca.
- **`resultJson`**: objeto do arquivo `result.json` do CLI; não `resultJSON`.
- `files`, `errorMessage`.

```bash
curl --fail-with-body http://localhost:3000/slice/status
curl --fail-with-body http://localhost:3000/slice/debug
```

Status/debug não têm isolamento por cliente ou correlação completa com `GET /jobs/{id}`. Cache hit pode não atualizar diagnóstico; falha antes de entrar no service pode não gerar debug novo. Compare timestamps antes de atribuir um erro à sua chamada.

As respostas contêm caminhos e conteúdo de perfis. Não as publique integralmente nem exponha a rota a clientes não confiáveis. Os logs estruturados HTTP/slicing incluem nomes, opções, duração e falhas do CLI; redija dados sensíveis antes de compartilhar.
