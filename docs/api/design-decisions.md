# Decisões de Design

## Profiles como JSON em disco

Decisão: manter profiles como arquivos `.json` em `DATA_PATH`.

Motivos:

- Simples de inspecionar e versionar
- Compatível com dashboards externos
- Evita banco de dados
- Facilita bind mount em Docker

## Overrides temporários

Decisão: overrides nunca alteram o profile salvo.

Motivos:

- Evita perda acidental de configuração
- Permite testar parâmetros por chamada
- Mantém profiles como fonte estável

## Apenas 1 slicing por vez

Decisão: lock global de slicing.

Motivos:

- OrcaSlicer CLI é pesado
- Hardware alvo pode ter RAM limitada
- Garante comportamento previsível

## Preview gera slicing real

Decisão: `/slice/preview` roda o OrcaSlicer de verdade.

Motivos:

- Detecção de suporte precisa refletir o que o Orca gerou
- Análise geométrica própria não seria 100% fiel
- G-code final é a fonte de verdade

## Thumbnail gerada pela API

Decisão: gerar thumbnail PNG fora do Orca CLI.

Motivos:

- Orca CLI/headless não gera thumbnail de forma confiável
- Xvfb não resolveu consistentemente
- Pós-processamento é controlável e previsível
- Formato atual é compatível com Neptune 4

## Remoção de Xvfb

Decisão: remover Xvfb do container.

Motivos:

- Não era necessário para o fluxo atual
- Aumentava complexidade do container
- Thumbnail agora é gerada pela API

## `enableSupport`, `brimType`, `printSequenceByObject`

Decisão: expor parâmetros de alto nível em vez de exigir overrides JSON.

Motivos:

- Facilita uso pelo dashboard
- Evita que frontend precise conhecer nomes internos do Orca
- Mantém compatibilidade com overrides avançados

## `resolveProfiles` separado de `sanitizeProfiles`

Decisão: flags separadas.

Motivos:

- Resolução lida com `inherits` e built-ins
- Sanitização lida com campos que quebram o CLI
- Permite diagnosticar problemas separadamente
