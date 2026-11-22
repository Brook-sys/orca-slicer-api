# Exemplos de integração

Base: `http://localhost:3000`. Arquivos e nomes são placeholders: cadastre os perfis conforme o [README](../../README.md). Use `--fail-with-body` e confira HTTP antes de interpretar um download como G-code. Nenhum exemplo valida impressão física.

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

## Slice síncrono com quantidade exata

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F quantity=3 -F arrange=true \
  -F sanitizeProfiles=true -F generateImage=true \
  -F enableSupport=false -F brimType=false \
  -D resultado.headers -o plate_1.gcode
```

Em sucesso, o corpo é somente G-code e os headers incluem `X-Slice-Quantity: 3`, `X-Slice-Plate: 1` e `Content-Type: application/x-gcode`. A API não devolve ZIP nem reduz a quantidade. Perfis nomeados são resolvidos mesmo sem `resolveProfiles=true`; sanitização não garante compatibilidade.

Rotação explícita usa graus e não pode ser combinada com `orient=true`. O eixo Z usa o campo **`rotate`**; X e Y usam `rotateX` e `rotateY`:

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F arrange=true -F rotate=90 \
  -o plate_1.gcode
```

Campos multipart desconhecidos não são rejeitados genericamente; o cliente deve usar os nomes documentados e conferir `/slice/debug` em ambiente de teste quando necessário.

## Quantidade que não cabe

Quando o CLI conclui com sucesso e a validação detecta que a quantidade ou as mesas divergiram, sync/preview retorna 422 com JSON semelhante ao exemplo abaixo. Se o próprio CLI falhar, a resposta é 500; no async, ambos aparecem como job `failed` após o 202:

```json
{"message":"requested quantity 12, but OrcaSlicer produced 4 object(s) on plate 1; reduce quantity and retry"}
```

Reduza `quantity` e reenvie. Não trate um arquivo parcial como sucesso: o contrato não o disponibiliza.

## Perfis diretos temporários

Os JSON precisam estar completos/planos: esse caminho não resolve `inherits`.

```bash
curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printerProfile=@printer_plano.json -F presetProfile=@preset_plano.json \
  -F filamentProfile=@filament_plano.json \
  -F 'overrides={"preset":{"layer_height":"0.2"}}' \
  -o plate_1.gcode
```

## Preview e async

```bash
curl --fail-with-body http://localhost:3000/slice/preview \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F quantity=3 -F arrange=true -o preview.json

curl --fail-with-body http://localhost:3000/slice/async \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F quantity=3 -F arrange=true
```

Inspecione `quantity`, `toolpaths.sourceComplete`, `truncated`, `omittedSegments`, `limits` e `warnings`. No default legacy, `layers` pode ser apenas um prefixo mesmo com a fonte completa analisada. Para pedir v2, repita o primeiro comando com `-F toolpathMode=full`; exija `version=2`, `geometryComplete=true` e nenhuma omissão. Full pode gerar dezenas de MB de JSON e retorna 422 em vez de geometria parcial quando excede seu orçamento. Veja [preview](preview.md).

Copie o `id` real da resposta async para `JOB_ID`:

```bash
curl --fail-with-body http://localhost:3000/jobs/JOB_ID
curl --fail-with-body http://localhost:3000/jobs/JOB_ID/download -o plate_1.gcode
curl --fail-with-body -X DELETE http://localhost:3000/jobs/JOB_ID
```

Baixe somente após `completed`. Se o job estiver `failed`, leia `error` e corrija a solicitação; `/download` retorna 400. O download entrega o único G-code validado, sem headers de metadados do sync. Cancelamento é tentativa, não exclusão do registro. Veja [slicing/jobs](slicing.md) e [preview](preview.md).
