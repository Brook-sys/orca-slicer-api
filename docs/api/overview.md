# Visão geral

A API Go recebe modelos e perfis, prepara arquivos temporários e executa o OrcaSlicer CLI. Não implementa o motor de fatiamento. A imagem é somente `linux/amd64`.

## Disponível no código

- CRUD de perfis em disco, import HTTPS e atualização manual da fonte.
- Aliases e resolução recursiva de perfis nomeados com overrides temporários.
- Slicing sync, jobs async, múltiplos modelos e export G-code/3MF.
- Preview por fatiamento real, estimativas e miniatura gerada pela API.
- Estado/debug persistidos, métricas JSON, OpenAPI e Swagger UI manuais.

Não há autenticação, isolamento por usuário, fila de espera, slicing paralelo ou recuperação durável de downloads após restart. Cache e listagem built-in têm limitações relevantes; consulte [operação](operations.md) e [perfis](profiles.md).

## Perfis

Nomeados (`printer`, `preset`, `filament`): busca em dados do usuário e JSON vendor, sempre resolvendo `inherits` na API, inclusive com `resolveProfiles=false`.

Diretos (`printerProfile`, `presetProfile`, `filamentProfile`): JSON multipart temporário, prioritário sobre o nome; não resolve herança. Envie perfis completos/planos.

## Evidência da atualização 2.5.0

AppImage inspecionado: 6 JSON + 72 caches `.opc` cereal internos. O upstream da mesma tag contém 14.813 JSON; o stage `vendorprofiles` fornece raiz separada `/app/profiles-json/resources/profiles`. O catálogo HTTP built-in ainda não percorre o layout vendor e retorna 404 nesse layout normal; resolução recursiva por nome é outra operação.

Houve sucesso de slice real no AppImage original 2.5.0 com três perfis achatados pela API: entrada STL de 297.684 bytes, 5.952 triângulos, CLI código 0, duração reportada de slice 1 segundo, 104 camadas, 340,83 mm de filamento, estimativa de impressão 6 min 52 s, G-code de 761.315 bytes e mesa a 60 °C.

A referência 2.4.0 fornecida tinha 75 camadas, 343,70 mm e estimativa 4 min 24 s. **Não houve equivalência de saída**, e não há validação de impressão física. Esse teste valida resolução pela API + execução do CLI, não o fluxo HTTP completo ou a imagem final.

Repetindo o STL com `arrange=false` (default), também houve código 0: 104 camadas, 341,04 mm, estimativa 6 min 50 s e mesa a 60 °C. Em teste sintético separado, um cubo posicionado na origem fora da área imprimível falhou com -102 sem arrange e passou com arrange. É uma restrição de posicionamento geométrico, não evidência de regressão geral do default. O default continua false.

Correção enviada para build; isso não é confirmação de build final, publicação concluída ou deploy. A validação ponta a ponta continua pendente. Propostas futuras ficam apenas no [roadmap](../ROADMAP.md).
