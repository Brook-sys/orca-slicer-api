# Slicing síncrono e jobs

## Contrato estrito de entrada e saída

`POST /slice`, `POST /slice/async` e `POST /slice/preview` aceitam **exatamente um STL** por chamada, no campo `file` ou no alias legado `files`. Enviar ambos, repetir arquivos ou usar STEP/3MF retorna 400. `quantity` representa o total de instâncias, incluindo o original, e deve estar entre 1 e 512; acima de 1, `arrange=true` é obrigatório.

A API executa o OrcaSlicer 2.5.0 com `--slice 1`, `--clone-objects N` e `--export-statistics`. O resultado só é aceito quando há:

- exatamente um G-code não vazio, chamado `plate_1.gcode`;
- `result.json` íntegro, com sucesso e somente a mesa 1 fatiada;
- `plate_1.statistics.json` versão 1, com exatamente `quantity` identidades `(name, instance)` válidas e únicas.

Depois de o CLI concluir com sucesso, mais de uma mesa/G-code ou contagem divergente retorna **422** no sync/preview e orienta reduzir `quantity` e tentar novamente. Se o próprio CLI falhar, a resposta é 500; no async, ambos aparecem como job `failed` após o 202. Evidência ausente, ilegível, duplicada ou de mesa incorreta falha fechado, normalmente com 500. Não há redução silenciosa, sucesso parcial, ZIP, 3MF ou G-code de outra mesa.

Essas provas verificam a contabilidade do resultado, **não detectam independentemente colisões geométricas**. O OrcaSlicer continua responsável por orientação, espaçamento, arranjo e fatiamento; revise o resultado antes de imprimir.

## `POST /slice`

Recebe multipart e retorna somente o G-code validado. Há um único lock de slicing **por processo**, compartilhado com async e preview. Se ocupado, retorna `409 {"message":"Slicer is busy"}` sem aguardar. Não há fila, paralelismo controlado ou coordenação entre réplicas.

A API grava o modelo como `model.stl` e os perfis em arquivos temporários, executa `ORCASLICER_PATH`, valida a saída, extrai metadados e opcionalmente injeta thumbnail. Perfis nomeados são sempre achatados, inclusive com `resolveProfiles=false`; uploads diretos não.

### Resposta 200

Download por `http.ServeFile`, com:

- `Content-Type: application/x-gcode`;
- `Content-Disposition: attachment; filename="plate_1.gcode"`;
- `X-Slice-Quantity`: quantidade exata validada;
- `X-Slice-Plate: 1`;
- `X-Print-Time-Seconds`: estimativa em segundos;
- `X-Filament-Used-g`: gramas estimadas;
- `X-Filament-Used-mm`: comprimento estimado em milímetros.

Valores dependem dos metadados disponíveis; zero não comprova ausência de consumo. O workdir sync é removido depois da resposta. O timeout padrão de 1800 segundos cobre o processo CLI, não todo o upload e pós-processamento.

## `POST /slice/async`

Usa a mesma entrada e as mesmas validações do sync. Cria job e devolve **202** com `id`, `status`, `progress`, `createdAt`, `updatedAt` e `error` opcional. O upload e a validação inicial ainda ocorrem antes da resposta. O processamento usa contexto independente da conexão do cliente.

Não é fila: a goroutine tenta o mesmo lock e pode falhar imediatamente se ocupado. Um 202 confirma criação do job, não conclusão nem reserva de capacidade. Overflow após o 202 aparece como `status=failed`, com a mensagem para reduzir `quantity`; o download não entrega G-code parcial.

| Rota | Comportamento |
|---|---|
| `GET /jobs` | Array dos metadados de todos os jobs conhecidos, sem paginação/ordenação garantida |
| `GET /jobs/{id}` | Estado atual; 404 se desconhecido |
| `GET /jobs/{id}/download` | Após `completed`, entrega o único `plate_1.gcode` como `application/x-gcode`; 400 se pendente/falho |
| `DELETE /jobs/{id}` | Tenta cancelar o processo ativo; 204 para job conhecido, inclusive já terminado; não apaga job |

O download async não inclui os headers de metadados/quantidade do sync. Estados de job: `pending`, `slicing`, `completed`, `failed`. Progresso é extraído de logs do CLI, pode ficar em 0 ou avançar irregularmente; completed define 100. Cancelamento aparece como failed, não como estado cancelled na API de jobs. Há janela antes do registro da função de cancelamento; 204 não garante interrupção imediata.

### Persistência e retenção

Metadados ficam em `DATA_PATH/jobs`. Jobs ativos recarregados após restart viram failed (`job lost after restart`). Caminhos de resultado e workdir não são serializados: jobs recarregados como completed também deixam de ter download disponível. Não há retomada do processo.

A cada hora o servidor remove registros completed/failed com mais de 24 horas. Isso não garante remoção de todos os workdirs temporários/artefatos; monitore disco. Veja [operação e cache](operations.md), [parâmetros](slice-parameters.md), [preview](preview.md) e [erros](errors.md).
