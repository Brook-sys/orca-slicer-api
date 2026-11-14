# Fluxos de integração

## Configuração e primeiro slice

1. Conferir health, executável e raiz JSON. Health 200 sozinho não prova resolução/slicing.
2. Cadastrar perfis compatíveis; preservar backup antes de sobrescrever.
3. Resolver nomes com `/slice/resolve-profiles`; mostrar warnings ao operador.
4. Enviar um modelo sintético com os três perfis selecionados.
5. Conferir HTTP, arquivo, estimativas e movimentos antes de imprimir.

Exemplos completos em [examples.md](examples.md).

## Preview opcional

Executa slice real, força suporte e imagem. Exiba `usesSupport` como observação dos marcadores do resultado, não como garantia de imprimibilidade. O thumbnail usa STL/primeiro modelo. Pode custar outro slice e bloquear a instância.

## Job async

1. Enviar multipart para `/slice/async`.
2. Receber 202/id, sem presumir que há capacidade reservada.
3. Fazer polling moderado de `/jobs/{id}` até completed/failed.
4. Após completed, baixar primeiro arquivo por `/download` e guardar no cliente.
5. Se solicitado, DELETE tenta cancelar; confirme estado posteriormente.

Não há fila de espera nem recuperação de resultados após restart. Não usar `/slice/status` como substituto do estado do job específico.

## Falhas e repetição

Não repetir automaticamente POST após timeout sem verificar o estado conhecido: pode duplicar trabalho. Em 409, informar ocupado e deixar o cliente controlar nova tentativa. Em erro de perfil, resolver primeiro e conferir raiz/pai/aliases. `/slice/debug` é global: comparar timestamps.

Quando variar overrides ou parâmetros para comparação, controlar o cache em ambiente isolado; a chave atual pode ignorar a variação. Não tratar um hit como comprovação de novo slice.
