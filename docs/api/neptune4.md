# Thumbnails e controles Neptune 4

`generateImage=true` ou `GENERATE_IMAGE=true` solicita pós-processamento do G-code pela API. A miniatura não é nativa do CLI: renderiza a geometria STL em PNG 160×160, fundo transparente, codifica em base64 e insere comentários `; thumbnail begin`/`; thumbnail end`. Se já existir esse marcador, não gera outro bloco.

No preview, a mesma renderização é retornada no campo JSON `thumbnail`, sem injeção no G-code analisado. No sync/async, a injeção usa arquivo temporário e cópia incremental em vez de carregar o G-code inteiro para reescrevê-lo.

## Limites

- O parser de miniatura aceita STL binário/ASCII; os endpoints rejeitam STEP/3MF e múltiplos modelos antes do CLI.
- A imagem usa a geometria do STL original, sem aplicar quantidade, rotação ou arranjo final. Não representa disposição da mesa, toolpath ou suporte.
- Somente `exportType=gcode` existe no contrato atual.
- O formato foi direcionado ao fluxo Neptune 4, mas não há garantia universal de exibição por firmware/tela nem impressão física validada nesta atualização.

## Controles de processo

| Multipart | Efeito no preset temporário |
|---|---|
| `enableSupport=true/false` | `enable_support` como string `"1"`/`"0"`, se presente |
| `brimType=true` | `brim_type=auto_brim` |
| `brimType=false` | `brim_type=no_brim` |
| `printSequenceByObject=true` | `print_sequence=by object` |
| Sequência ausente/false | `print_sequence=by layer` |

Suporte e brim ausentes preservam o valor do processo. Esses controles só podem atuar sobre um preset selecionado/enviado. Revise colisões e folgas antes de imprimir por objeto; a API não valida segurança mecânica.

Veja [parâmetros](slice-parameters.md), [preview](preview.md) e [exemplos genéricos](examples.md).
