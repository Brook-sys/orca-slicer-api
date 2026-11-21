# Visão geral

A API Go recebe um STL e perfis, prepara arquivos temporários e executa o OrcaSlicer CLI. Não implementa o motor de fatiamento. A imagem é somente `linux/amd64`.

## Disponível no código

- CRUD de perfis em disco, import HTTPS e atualização manual da fonte.
- Aliases e resolução recursiva de perfis nomeados com overrides temporários.
- Slicing sync/async de **exatamente um STL**, quantidade de 1 a 512 e somente G-code da mesa 1.
- Validação de `plate_1.gcode`, manifesto e estatísticas, sem redução de quantidade, mesas extras, ZIP ou 3MF.
- Preview por fatiamento real, estimativas, miniatura e toolpaths limitados extraídos do G-code.
- Cache com snapshots de evidência e revalidação do contrato de saída em cada hit, sem hash do snapshot ou validação semântica do G-code.
- Estado/debug persistidos, métricas JSON, OpenAPI e Swagger UI manuais.

Não há autenticação, isolamento por usuário, fila de espera, slicing paralelo ou recuperação durável de downloads após restart. Consulte [operação](operations.md) e [perfis](profiles.md).

## Perfis

Nomeados (`printer`, `preset`, `filament`): busca em dados do usuário e JSON vendor, sempre resolvendo `inherits` na API, inclusive com `resolveProfiles=false`.

Diretos (`printerProfile`, `presetProfile`, `filamentProfile`): JSON multipart temporário, prioritário sobre o nome; não resolve herança. Envie perfis completos/planos.

O AppImage inspecionado contém 6 JSON e 72 caches `.opc` internos. O upstream da mesma tag contém 14.813 JSON; o stage `vendorprofiles` fornece a raiz separada `/app/profiles-json/resources/profiles`. O catálogo HTTP built-in não percorre o layout vendor normal e pode retornar 404; resolução recursiva por nome é outra operação.

## Evidência de execução real 2.5.0

Foi executado o AppImage oficial OrcaSlicer 2.5.0, SHA-256 `d13eb0a9cd618ed080ab6b82c64c0a60c4501aac91e88d5ac2c3b526fb141a7c`, associado ao source commit `40ecc66f78ed72dc368e0c1ebe40d77f00cc8a19`, com perfis de teste e STLs sintéticos. A validação HTTP local cobriu:

- sync e async com três cubos, três identidades no mesmo `plate_1.gcode`, `application/x-gcode` e nenhum ZIP;
- quatro peças que couberam exatamente na mesa e overflow de doze, no qual o Orca produziu quatro: sync retornou 422 e async terminou failed, ambos sem download parcial;
- preview de três cubos com 33 camadas, 1.029 segmentos, três identidades, fonte completa e sem truncamento;
- rotações reais X/Y/Z com dimensões compatíveis com a permuta dos eixos dentro da tolerância de 0,45 mm do teste e sem perda da geometria abaixo de Z=0 após `--ensure-on-bed`;
- suporte real com 2.095 segmentos de suporte e brim separado com 295 segmentos;
- caso com 21.834 segmentos, dos quais 20.000 foram retidos e 1.834 reportados como omitidos, mantendo `sourceComplete=true`;
- rejeições 400 para quantidade inválida, múltiplos STLs, mesa adicional, 3MF, espaçamento positivo e `orient` com rotação.

Regressões automatizadas também cobrem cache sem manifesto/estatísticas, mesa errada, identidades duplicadas, G-code vazio, G92, bounds após truncamento e orçamento global de arcos.

Esses resultados são evidência de compatibilidade dos caminhos testados, **não prova de ausência de colisões, impressão física segura, equivalência com 2.4.0 ou robustez sob produção**. A contagem exata e uma única mesa não substituem inspeção geométrica. Nenhuma impressora foi acionada.

A validação HTTP local não comprova imagem final de contêiner, publicação ou deploy; essas etapas exigem evidência própria. Propostas futuras ficam no [roadmap](../ROADMAP.md).
