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

## Contrato estrito e rotação

A API entrega somente um G-code da mesa 1 e valida manifesto/estatísticas contra a quantidade pedida. Não converte overflow em sucesso parcial. Rotações explícitas acrescentam `--ensure-on-bed` porque o CLI 2.5.0 centraliza o volume na origem após cada rotação; `orient=true` é incompatível com elas.

## Preview e thumbnail

Preview executa slice real sem alterar a decisão de suporte. O PNG é gerado separadamente do STL original; o G-code não recebe injeção nesse fluxo e é lido pelo parser de toolpaths. Sync/async com `generateImage=true` injetam o PNG no G-code por cópia incremental. A miniatura não representa arranjo final, suportes ou toolpath.

## Defaults não mudados por um teste

`arrange` continua false. Um cubo na origem fora da área imprimível precisou de arrange, mas o STL de 5.952 triângulos passou também sem arrange: não é regressão geral do default. Estimativas e camadas diferentes entre versões não devem ser descritas como equivalentes. Não há validação física de impressão nesta atualização.

Novas propostas ficam no [roadmap](../ROADMAP.md), separadas do comportamento atual.
