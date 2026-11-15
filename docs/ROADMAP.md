# Roadmap (planejamento)

As propostas abaixo **não estão implementadas**, não têm prazo/compromisso de entrega e não devem ser anunciadas como funcionalidades atuais. Exigem escopo autorizado, projeto e testes antes de qualquer implementação.

## Pendências de validação da correção 2.5.0

- Validar fluxo HTTP completo e imagem final de revisão conhecida, incluindo perfis, metadados, thumbnails, preview e jobs.
- Comparar saída entre versões sem presumir equivalência; avaliar impressão física separadamente, com supervisão.
- Confirmar build/publicação e qualquer atualização de runtime por evidência própria; um push não é deploy.

## Lacunas conhecidas a avaliar

- Catálogo built-in recursivo por fabricante/categoria: hoje a rota rasa retorna 404 no layout vendor normal, apesar da resolução por nome funcionar. Expansão não faz parte desta correção.
- Evoluir cache para capturar recursos implícitos/ambiente do CLI e medir custo de hash/cópias; settings, perfis efetivos, executável e ownership de snapshots já são considerados.
- Entrega durável de resultados de jobs após restart, múltiplos arquivos async e coleta de temporários.
- Autenticação/autorização, endurecimento SSRF, limites agregados de upload, timeouts HTTP e quotas de recursos.
- Contrato OpenAPI alinhado aos handlers e testes de divergência.
- Controle explícito de capacidade/fila, somente após requisitos de carga e política de cancelamento definidos.
- Thumbnails para outros formatos e mesa completa, com limites de memória definidos.
- ARM64 real: seleção/validação de executável nativo e runtime compatível; imagem atual é amd64-only.

Não implementar essas propostas durante tarefas de documentação ou correção com escopo menor. Consulte [operação](api/operations.md) para riscos atuais, que continuam existindo até validação de mudanças específicas.
