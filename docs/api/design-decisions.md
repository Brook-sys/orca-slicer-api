# Decisões e consequências atuais

## Go stdlib e arquivos em disco

O serviço usa Go 1.23 sem dependências externas ou banco de dados. JSON em disco facilita inspeção/backup, mas não fornece transações multi-arquivo, coordenação entre processos ou isolamento multiusuário.

## Um slice por instância

O mutex limita execução simultânea do CLI. Async desacopla a conexão, não implementa fila: pode falhar por ocupado após 202. Não anunciar escalabilidade ou paralelismo garantidos.

## Resolução antes do CLI 2.5.0

Perfis nomeados são achatados antes do CLI, removendo `inherits` e aplicando overrides. A flag `resolveProfiles` permanece compatível no contrato, mas não desliga esse fluxo. JSON upstream da mesma tag fornece a herança ausente nos caches internos do AppImage. Recursos originais do AppImage são preservados.

Fallback por tokens é último recurso, gera warning e rejeita ambiguidades; não significa equivalência semântica. Sanitização é separada: define `from=system` e remove `small_perimeter_speed` de processos. Não é validador geral.

## Perfis temporários e controles

Overrides e controles alteram cópias temporárias, não perfis persistidos. Perfis multipart têm precedência sobre nomes e não são resolvidos recursivamente. Printer e preset precisam estar presentes juntos para formar `--load-settings`.

## Preview e thumbnail

Preview executa slice real e força suporte/imagem. A API gera PNG de STL e injeta no G-code, sem depender de renderização gráfica do CLI. Isso não cobre geometria STEP/3MF nem mesa completa com múltiplos modelos.

## Defaults não mudados por um teste

`arrange` continua false. Um cubo na origem fora da área imprimível precisou de arrange, mas o STL de 5.952 triângulos passou também sem arrange: não é regressão geral do default. Estimativas e camadas diferentes entre versões não devem ser descritas como equivalentes. Não há validação física de impressão nesta atualização.

Novas propostas ficam no [roadmap](../ROADMAP.md), separadas do comportamento atual.
