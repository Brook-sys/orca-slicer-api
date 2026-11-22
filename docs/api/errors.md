# Erros e diagnóstico

Erros dos handlers usam `Content-Type: application/json`:

```json
{"message":"Slicer is busy"}
```

O campo é `message`, não `error`. Respostas de roteamento do Go (404/405 etc.) podem ser texto, e downloads interrompidos podem não permitir substituir a resposta por JSON. Em jobs async, a falha posterior ao 202 aparece no campo `error` do job.

| HTTP | Situação |
|---|---|
| 400 | Multipart/JSON inválido, STL ausente ou múltiplo, STEP/3MF, limite por arquivo, categoria/nome inválidos, `quantity` fora de 1–512, mesa/export/`toolpathMode` inválidos, espaçamento positivo ou `orient` combinado com rotação |
| 404 | Perfil/job/alias não encontrado; listagem built-in sem diretório suportado |
| 408 | `Slicing cancelled or timed out` durante execução CLI |
| 409 | `Slicer is busy`, ou import sem overwrite para perfil existente |
| 422 | Após sucesso do CLI, cópias/mesas não couberam no resultado estrito, a contagem de objetos divergiu ou o preview full excedeu 4.096 camadas/250.000 segmentos; nenhum resultado geométrico parcial é entregue |
| 500 | CLI, filesystem, download externo, thumbnail, parser ou evidência de saída ausente/inválida |
| 503 | Health essencial falhou |

## Quantidade e integridade da saída

Exemplo real de overflow:

```json
{"message":"requested quantity 12, but OrcaSlicer produced 4 object(s) on plate 1; reduce quantity and retry"}
```

A mensagem efetiva contém os números solicitado/produzido. Depois de o CLI concluir com sucesso, mais de um G-code ou mais de uma mesa também produz 422 com orientação para reduzir `quantity`. Se o próprio CLI falhar, a resposta é 500; no async, ambos aparecem como job `failed` após o 202. Nenhum G-code parcial é entregue.

São falhas de integridade, normalmente 500, e não overflow tratável pelo usuário: ausência/corrupção de `result.json` ou `plate_1.statistics.json`, G-code vazio, nome diferente de `plate_1.gcode`, identidade inválida/duplicada ou manifesto sem sucesso. Um snapshot de cache que não passe novamente por todas essas validações é descartado como miss e o serviço tenta um novo fatiamento; ele não é servido como resultado parcial.

## Perfis 2.5.0

`parent profile was not found`: conferir a raiz `ORCA_PROFILES_PATH` configurada, disponibilidade do pai e aliases. Ligar `resolveProfiles` não corrige uma raiz errada: nomes já são sempre resolvidos.

`The input preset file is invalid and can not be parsed`: inspecionar perfis finais e saída do CLI. Para JSON multipart, envie perfis planos; esse caminho não resolve herança. Sanitização só ajusta `from` e remove um campo conhecido do processo; não é correção universal.

Warnings de fallback de nomes exigem revisão. Fallback por tokens ambíguo é rejeitado, não escolhe arbitrariamente um candidato. Configure um alias explícito ou use nome inequívoco após conferir os perfis.

## Posicionamento e rotação

`arrange=false` preserva a responsabilidade do modelo/perfil pelo posicionamento; peças fora da área imprimível podem ser rejeitadas pelo Orca. `arrange=true` é obrigatório para múltiplas cópias, mas não corrige toda geometria inválida.

No Orca 2.5.0, cada `--rotate*` centraliza o volume na origem. A API acrescenta `--ensure-on-bed` quando há rotação explícita para evitar omitir a metade abaixo de Z=0. `orient=true` com rotação explícita é rejeitado, pois a orientação automática substitui essas rotações.

No preview, fonte acima de 32 MiB, linha acima de 1 MiB, mais de 4.000.000 de linhas, comando não suportado ou excesso de trabalho de arcos retorna 500 sem prévia parcial. Em legacy, atingir somente os limites de retenção de 512 camadas/20.000 segmentos não é erro: a resposta traz `truncated=true` e `omittedSegments`.

`toolpathMode` diferente de `legacy`/`full` retorna 400 antes do fatiamento. Em full, exceder 4.096 camadas ou 250.000 segmentos retorna 422 com uma das mensagens abaixo e sem JSON de preview parcial:

```json
{"message":"Full toolpath preview exceeds the 250000-segment retention limit; no partial geometry returned"}
```

A falha de camadas usa `Full toolpath preview exceeds the 4096-layer retention limit; no partial geometry returned`. O consumidor deve tratar o status e `message`, não depender do texto completo.

## Sequência de diagnóstico

1. Registrar HTTP e `message` sem confundir corpo JSON de erro com G-code.
2. Conferir `/health`, versão efetiva e raiz JSON.
3. Inspecionar `/profiles/resolve`: resultado e warnings.
4. Consultar `/slice/debug`: `output`, `slicerError`, `resultJson.error_string` e perfis finais. O output retido é limitado a 1 MiB.
5. Reproduzir com modelo sintético e cache de teste controlado. Não publicar logs ou perfis pessoais.
