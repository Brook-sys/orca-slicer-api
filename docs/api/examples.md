# Exemplos de integração

Base: `http://localhost:3000`. Arquivos e nomes são placeholders: cadastre os perfis conforme o [README](../../README.md). Nenhum exemplo comprova validação 2.5.0 completa. Use `--fail-with-body` e confira HTTP antes de interpretar um download como G-code.

## Diagnóstico e perfis

```bash
curl --fail-with-body http://localhost:3000/health
curl --fail-with-body http://localhost:3000/profiles/printers
curl --fail-with-body http://localhost:3000/profiles/presets/upload \
  -F file=@preset.json -F name=meu_processo
curl --fail-with-body http://localhost:3000/profiles/resolve \
  -H 'Content-Type: application/json' \
  -d '{"category":"presets","name":"meu_processo","overrides":{}}'
```

## Slice síncrono

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F sanitizeProfiles=true -F generateImage=true \
  -F enableSupport=false -F brimType=false \
  -D resultado.headers -o resultado.gcode
```

Perfis nomeados são resolvidos mesmo sem `resolveProfiles=true`. Sanitização não garante compatibilidade. Reveja warnings no endpoint de resolução.

## Múltiplos modelos

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F files=@parte_a.stl -F files=@parte_b.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F arrange=true -o resultado.bin
```

Verifique `Content-Disposition`: vários arquivos de saída retornam ZIP, não necessariamente um único G-code.

## Perfis diretos temporários

Os JSON precisam estar completos/planos: esse caminho não resolve `inherits`.

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printerProfile=@printer_plano.json -F presetProfile=@preset_plano.json \
  -F filamentProfile=@filament_plano.json \
  -F 'overrides={"preset":{"layer_height":"0.2"}}' \
  -o resultado.gcode
```

## Preview e async

```bash
curl --fail-with-body http://localhost:3000/slice/preview \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento

curl --fail-with-body http://localhost:3000/slice/async \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento
```

Copie o `id` real da resposta para `JOB_ID`:

```bash
curl --fail-with-body http://localhost:3000/jobs/JOB_ID
curl --fail-with-body http://localhost:3000/jobs/JOB_ID/download -o resultado.gcode
curl --fail-with-body -X DELETE http://localhost:3000/jobs/JOB_ID
```

Baixe somente após completed. Download entrega o primeiro arquivo, sem headers de metadados do sync. Cancelamento é tentativa, não exclusão do registro. Veja [slicing/jobs](slicing.md).
