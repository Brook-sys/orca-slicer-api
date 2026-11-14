# Thumbnails e controles Neptune 4

`generateImage=true` ou `GENERATE_IMAGE=true` solicita pós-processamento do G-code pela API. A miniatura não é nativa do CLI: renderiza a geometria STL em PNG 160×160, fundo transparente, codifica em base64 e insere comentários `; thumbnail begin`/`; thumbnail end`. Se já existir esse marcador, não gera outro bloco.

## Limites

- O parser de miniatura aceita STL binário/ASCII. Aceitar STEP/3MF no endpoint de slicing **não** significa que a geração de imagem suporte esses formatos; pode falhar após o CLI concluir.
- Em múltiplos modelos, usa a geometria do **primeiro** modelo para todos os arquivos de saída. Não representa disposição final da mesa, toolpath ou suporte.
- `exportType=3mf` não recebe essa injeção de thumbnail.
- O formato foi direcionado ao fluxo Neptune 4, mas não há garantia universal de exibição por firmware/tela nem impressão física validada nesta atualização.

## Controles de processo

| Multipart | Efeito no preset temporário |
|---|---|
| `enableSupport=true/false` | `enable_support` bool, se presente |
| `brimType=true` | `brim_type=auto_brim` |
| `brimType=false` | `brim_type=no_brim` |
| `printSequenceByObject=true` | `print_sequence=by object` |
| Sequência ausente/false | `print_sequence=by layer` |

Suporte e brim ausentes preservam o valor do processo. Esses controles só podem atuar sobre um preset selecionado/enviado. Revise colisões e folgas antes de imprimir por objeto; a API não valida segurança mecânica.

Veja [parâmetros](slice-parameters.md), [preview](preview.md) e [exemplos genéricos](examples.md).
