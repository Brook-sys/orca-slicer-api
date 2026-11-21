# Fluxos de integração

## Configuração e primeiro slice

1. Conferir health, executável e raiz JSON. Health 200 sozinho não prova resolução/slicing.
2. Cadastrar perfis compatíveis; preservar backup antes de sobrescrever.
3. Resolver nomes com `/slice/resolve-profiles`; mostrar warnings ao operador.
4. Enviar um STL sintético com os três perfis selecionados; para múltiplas cópias, usar `quantity` e `arrange=true`.
5. Conferir HTTP 200, `X-Slice-Quantity`, `X-Slice-Plate: 1`, G-code, estimativas e movimentos antes de imprimir. Em 422, reduzir quantidade; não procurar um ZIP ou saída parcial.

Exemplos completos em [examples.md](examples.md).

## Preview opcional

Executa slice real e preserva a decisão de suporte do pedido/preset. Exiba `usesSupport` como observação dos movimentos de suporte no G-code, não como garantia de imprimibilidade. O thumbnail usa o STL original; os toolpaths vêm do G-code. Apresente `sourceComplete`, `truncated`, `omittedSegments`, `limits` e `warnings`. Pode custar outro slice e bloquear a instância.

## Job async

1. Enviar multipart para `/slice/async`.
2. Receber 202/id, sem presumir que há capacidade reservada.
3. Fazer polling moderado de `/jobs/{id}` até completed/failed.
4. Após completed, baixar o único `plate_1.gcode` por `/download` e guardar no cliente; failed não tem download parcial.
5. Se solicitado, DELETE tenta cancelar; confirme estado posteriormente.

Não há fila de espera nem recuperação de resultados após restart. Não usar `/slice/status` como substituto do estado do job específico.

## Falhas e repetição

Não repetir automaticamente POST após timeout sem verificar o estado conhecido: pode duplicar trabalho. Em 409, informar ocupado e deixar o cliente controlar nova tentativa. Em erro de perfil, resolver primeiro e conferir raiz/pai/aliases. `/slice/debug` é global: comparar timestamps.

Quando variar overrides ou parâmetros para comparação, controlar o cache em ambiente isolado; a chave inclui settings, perfis efetivos e executável, mas não captura ambiente/recursos implícitos do CLI. Não tratar um hit como comprovação de novo slice.
