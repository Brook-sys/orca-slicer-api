# Preview

`POST /slice/preview` recebe o mesmo multipart estrito de `/slice`: **um STL**, perfis/controles, quantidade exata e somente mesa 1. Executa fatiamento real pelo mesmo lock de sync/async; não é apenas renderização.

`enableSupport` ausente mantém o preset; true/false respeita o pedido. O preview não força suporte. A miniatura PNG 160×160 é gerada separadamente a partir do STL original e `generateImage` é desligado internamente, evitando alterar o G-code antes da análise. O toolpath vem do G-code fatiado real, não do thumbnail.

## Modos de toolpath

O multipart aceita `toolpathMode=legacy` ou `toolpathMode=full` somente neste endpoint. Ausente/`legacy` preserva o contrato v1 e o prefixo de 20.000 segmentos. `full` seleciona v2 e exige geometria completa dentro de 250.000 segmentos e 4.096 camadas; acima disso retorna 422 sem JSON de preview parcial. Outros valores retornam 400 antes do fatiamento.

```bash
curl --fail-with-body http://localhost:3000/slice/preview \
  -F file=@modelo.stl \
  -F printer=minha_impressora -F preset=meu_processo -F filament=meu_filamento \
  -F toolpathMode=full -o preview-completo.json
```

## Resposta

Exemplo estrutural abreviado no default **legacy/v1**:

```json
{
  "usesSupport": false,
  "printTime": 200,
  "filamentUsedG": 1.5853,
  "filamentUsedMm": 531.53,
  "thumbnail": "BASE64_DO_PNG_SEM_PREFIXO_DATA_URI",
  "quantity": 3,
  "toolpaths": {
    "version": 1,
    "positionUnit": "mm",
    "extrusionUnit": "mm",
    "feedrateUnit": "mm/s",
    "coordinateSystem": "gcode",
    "sourceComplete": true,
    "truncated": false,
    "sourceBytes": 137269,
    "sourceLines": 4000,
    "totalLayers": 33,
    "totalSegments": 1029,
    "returnedLayers": 33,
    "returnedSegments": 1029,
    "omittedSegments": 0,
    "supportSegments": 0,
    "brimSegments": 0,
    "layerSemantics": "markers",
    "warnings": [],
    "limits": {
      "sourceBytes": 33554432,
      "lineBytes": 1048576,
      "lines": 4000000,
      "layers": 512,
      "segments": 20000,
      "arcSegmentsPerMove": 4096,
      "arcSegmentsTotal": 1000000,
      "arcChordToleranceMm": 0.05
    },
    "bounds": {"min": [114.2, 114.2, 0.3], "max": [135.8, 135.8, 9.9]},
    "layers": [
      {
        "index": 0,
        "z": 0.3,
        "segments": [
          {
            "from": [120.2, 120.2, 0.3],
            "to": [121.0, 120.2, 0.3],
            "extrusion": 0.02,
            "feedrateMmSec": 20,
            "feature": "Outer wall",
            "object": "model.stl",
            "tool": 0
          }
        ]
      }
    ]
  }
}
```

Os números são ilustrativos. `printTime` está em segundos, massa em gramas, comprimento e coordenadas em milímetros; feedrate em mm/s. Estimativas não são medições físicas.

No modo **full/v2**, o envelope acrescenta `mode` e `geometryComplete`, eleva `version` e os limites de retenção, e cada segmento contém explicitamente `widthMm` e `heightMm`:

```json
{
  "toolpaths": {
    "version": 2,
    "mode": "full",
    "geometryComplete": true,
    "sourceComplete": true,
    "truncated": false,
    "totalLayers": 240,
    "returnedLayers": 240,
    "totalSegments": 76160,
    "returnedSegments": 76160,
    "omittedSegments": 0,
    "limits": {"layers": 4096, "segments": 250000},
    "layers": [{
      "index": 0,
      "z": 0.28,
      "segments": [{
        "from": [120.2, 120.2, 0.28],
        "to": [121.0, 120.2, 0.28],
        "extrusion": 0.02,
        "feedrateMmSec": 20,
        "feature": "Bridge",
        "object": "model.stl",
        "tool": 0,
        "widthMm": 0.42,
        "heightMm": 0.28
      }]
    }]
  }
}
```

Os demais campos obrigatórios do envelope foram abreviados apenas neste segundo exemplo. `widthMm`/`heightMm` são `null` quando o Orca não declarou o valor vigente; não há inferência de largura/altura. O legacy não ganha `mode`, `geometryComplete`, `widthMm` ou `heightMm`.

## Semântica dos toolpaths

- No legacy, só entram movimentos XY com **extrusão positiva**. Full retém extrusão positiva XYZ, inclusive movimento somente em Z. Nos dois modos, viagens sem extrusão, retracts, movimentos só de E e comentários isolados não viram segmentos.
- `feature` vem de `TYPE`/`FEATURE` e é preservado verbatim, inclusive `Bridge`, `Internal Bridge`, `Overhang wall`, infill e categorias de suporte; `object` vem dos comentários de objeto do Orca. Entre `WIPE_START` e `WIPE_END`, o segmento retornado usa `feature="Wipe"`; os contadores continuam baseados no `TYPE`/`FEATURE` vigente. Suporte contabiliza `Support`, `Support interface` e `Support transition`; brim contabiliza `Brim`.
- Em full, `widthMm` e `heightMm` seguem os últimos comentários `WIDTH:`/`HEIGHT:` ativos. Cada chave aparece em todo segmento, com número positivo em mm ou `null`. Valores inválidos falham fechado; no legacy esses comentários continuam não interpretados.
- `usesSupport` equivale a `supportSegments > 0` no G-code completo analisado. False não prova segurança sem suporte.
- Em legacy, `totalSegments`, `supportSegments`, `brimSegments` e `bounds` cobrem toda a fonte analisada, inclusive segmentos omitidos da resposta. `bounds` usa os pontos da geometria linearizada, não o volume sólido nem colisões. Em full, uma resposta 200 não tem geometria suportada omitida.
- Em legacy, `layers` e seus `segments` são um **prefixo retido**: até 512 camadas e 20.000 segmentos no total. `truncated=true` informa omissão; `omittedSegments` quantifica segmentos não retornados. Isso pode ocorrer com `sourceComplete=true`. Full retorna toda a geometria dentro dos tetos de 4.096 camadas/250.000 segmentos, com `truncated=false` e `omittedSegments=0`, ou falha sem preview parcial.
- `sourceComplete=true` só aparece depois da leitura completa dentro dos limites. Não há sucesso parcial quando a fonte excede um limite duro, contém comando não suportado ou estoura o trabalho de arcos. `geometryComplete=true` restringe-se à extrusão positiva XYZ suportada: não inclui travels, retracts, expansão de macros, colisões nem validação física de impressão.
- `P1` é aceito somente em G2/G02/G3/G03, incluindo elevação helicoidal emitida pelo Orca; não acrescenta outra volta. Outros valores de P falham fechado por diferenças entre dialetos. Blocos PNG `thumbnail begin …` / `thumbnail end` são reconhecidos e validados estruturalmente; não são interpretados como comandos. Veja a [validação com modelos reais](model-corpus-validation.md).
- Arcos G2/G3 no plano XY são linearizados com tolerância de corda de 0,05 mm, até 4.096 segmentos por movimento e orçamento global de 1.000.000 segmentos de tesselação com extrusão. O endpoint preserva o endpoint exato e distribui a extrusão.
- G20/G21, G90/G91, M82/M83 e G92 são interpretados. Homing é assumido em zero para visualização. Macros de impressora e comandos Klipper de dinâmica não são expandidos; `warnings` registra essas limitações. `SET_RETRACTION` e `SET_PRESSURE_ADVANCE` são reconhecidos como configuração de firmware: seus parâmetros não são coordenadas e seus efeitos não são simulados. Isso não valida parâmetros para execução no firmware. Comandos desconhecidos, inclusive transformações como `SET_GCODE_OFFSET`, continuam falhando fechado; não há descarte genérico de macros.

## Limites duros

| Recurso | Legacy/v1 | Full/v2 |
|---|---:|---:|
| G-code de origem | 32 MiB (33.554.432 bytes) | Igual |
| Linha | 1 MiB (1.048.576 bytes) | Igual |
| Linhas lidas | 4.000.000 | Igual |
| Camadas retidas | 512; omite excedente | 4.096; excedente retorna 422 |
| Segmentos retidos | 20.000; omite excedente | 250.000; excedente retorna 422 |
| Segmentos por arco | 4.096 | Igual |
| Trabalho global de arcos | 1.000.000 | Igual |

Esses limites protegem a extração do toolpath; **não limitam a complexidade do STL, o uso de CPU/RAM do Orca, a geração do thumbnail ou o tamanho agregado do upload**. O modelo continua sujeito ao teto individual de 100.000.000 bytes. Fontes acima de 32 MiB são rejeitadas, não truncadas silenciosamente.

O preview pode custar um slice completo e não substitui revisão do G-code. Cache pode evitar o CLI, mas o G-code ainda é analisado para montar a resposta. Pode retornar 400 por entrada ou `toolpathMode` inválido, 409 ocupado, 408 timeout/cancelamento do CLI, 422 por quantidade/mesa ou orçamento full excedido e 500 por CLI, imagem, evidência ou parser. Exemplos em [examples.md](examples.md) e limites operacionais em [operations.md](operations.md).

O formato full continua sendo JSON de objetos, não arrays compactados. No teto sintético de 250.000 segmentos foram medidos **38.746.814 bytes**; uma requisição real com 228.444 segmentos no teto final produziu **42.253.033 bytes** e pico observado de **259.572 KiB de RSS no processo da API** (uma execução anterior da mesma geometria atingiu 283.664 KiB). Essas amostras não estabelecem memória máxima do contêiner/Orca nem segurança sob concorrência. O frontend precisa aceitar explicitamente v2 e impor limites de memória/geometria antes de renderizar; o download e o parse JSON já podem ser custosos. Consulte as [medições com modelos reais](model-corpus-validation.md).
