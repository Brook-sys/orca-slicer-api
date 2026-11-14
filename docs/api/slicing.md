# Slicing síncrono e jobs

## `POST /slice`

Recebe multipart e retorna um arquivo produzido pelo CLI ou ZIP quando há vários resultados. Consulte [parâmetros](slice-parameters.md) e [exemplos](examples.md).

Há um único lock de slicing **por processo**, compartilhado com async e preview. Se ocupado, retorna `409 {"message":"Slicer is busy"}` sem aguardar. Não há fila, paralelismo controlado ou coordenação entre réplicas.

A API grava modelos/perfis temporários, executa `ORCASLICER_PATH`, coleta `.gcode` ou `.3mf`, extrai metadados e opcionalmente injeta thumbnail. Perfis nomeados são sempre achatados, inclusive com `resolveProfiles=false`; uploads diretos não.

### Resposta

Um arquivo: download por `http.ServeFile`, com `Content-Disposition`; MIME pode depender da detecção do Go/extensão. Vários arquivos: `application/zip`, `result.zip`. Não assuma um MIME fixo para G-code/3MF.

Headers de metadados do sync:

- `X-Print-Time-Seconds`: estimativa em segundos.
- `X-Filament-Used-g`: gramas estimadas.
- `X-Filament-Used-mm`: comprimento estimado em milímetros.

Valores dependem dos metadados disponíveis no resultado; zero não comprova ausência de consumo. Workdir do resultado sync é removido depois da resposta. O timeout padrão de 1800 segundos cobre o processo CLI, não toda a requisição, upload e pós-processamento.

## `POST /slice/async`

Mesma entrada multipart do sync. Cria job e devolve **202** com `id`, `status`, `progress`, `createdAt`, `updatedAt` e `error` opcional. O upload e a leitura ainda ocorrem antes da resposta. O processamento usa contexto independente da conexão do cliente.

Não é fila: a goroutine tenta o mesmo lock e pode falhar imediatamente se ocupado. Um 202 confirma criação do job, não conclusão nem reserva de capacidade.

| Rota | Comportamento |
|---|---|
| `GET /jobs` | Array dos metadados de todos os jobs conhecidos, sem paginação/ordenação garantida |
| `GET /jobs/{id}` | Estado atual; 404 se desconhecido |
| `GET /jobs/{id}/download` | Só após completed com resultado; 400 caso contrário; entrega **apenas primeiro arquivo**, não ZIP |
| `DELETE /jobs/{id}` | Tenta cancelar o processo ativo; 204 para job conhecido, inclusive já terminado; não apaga job |

Estados de job: `pending`, `slicing`, `completed`, `failed`. Progresso é extraído de logs do CLI, pode ficar em 0 ou avançar irregularmente; completed define 100. Cancelamento aparece como failed, não como estado cancelled na API de jobs. Há janela antes do registro da função de cancelamento; 204 não garante interrupção imediata.

### Persistência e retenção

Metadados ficam em `DATA_PATH/jobs`. Jobs ativos recarregados após restart viram failed (`job lost after restart`). Caminhos de resultado e workdir não são serializados: até jobs recarregados como completed deixam de ter download disponível. Não há retomada do processo.

A cada hora o servidor remove registros completed/failed com mais de 24 horas. Isso não garante remoção de todos os workdirs temporários/artefatos; monitore disco. Veja [operação e cache](operations.md).
