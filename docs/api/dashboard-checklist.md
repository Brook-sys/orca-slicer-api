# Checklist de cliente/dashboard

Esta lista orienta uma integração externa; não descreve um frontend fornecido pelo projeto.

- [ ] Configurar URL e acesso por rede confiável/proxy autenticado. CORS não autentica.
- [ ] Conferir health sem confundi-lo com teste de slice.
- [ ] Carregar listas de perfis salvos, com source/checksum; não depender do catálogo built-in, que retorna 404 no layout vendor normal.
- [ ] Exigir `name` ASCII válido no upload; avisar que upload sobrescreve.
- [ ] Oferecer import/update apenas a operadores confiáveis; há risco SSRF.
- [ ] Resolver perfis nomeados e exibir warnings; não oferecer `resolveProfiles=false` como bypass de herança.
- [ ] Exigir perfis completos/planos nos uploads diretos de slice.
- [ ] Aceitar somente um STL e `application/x-gcode`; diferenciar corpo JSON de erro (`message`) e nunca esperar ZIP/3MF.
- [ ] Enviar `quantity` entre 1 e 512 e `arrange=true` acima de 1; em 422, pedir redução de quantidade em vez de usar resultado parcial.
- [ ] Exibir unidades: segundos, gramas e milímetros; estimativas não são medições reais.
- [ ] Avisar que preview roda CLI, preserva a decisão de suporte e não garante necessidade física de suporte.
- [ ] Exibir `quantity`, `sourceComplete`, `truncated`, `omittedSegments`, `limits` e `warnings`; não apresentar o prefixo retido como toolpath integral.
- [ ] Não aceitar múltiplos STLs, STEP/3MF nem interpretar thumbnail como a disposição final da mesa.
- [ ] Tratar 409 e async failed por ocupado; não mostrar fila inexistente.
- [ ] Usar job específico para polling; guardar download antes de reiniciar servidor.
- [ ] Apresentar cancelamento como tentativa e conferir estado depois.
- [ ] Não publicar conteúdo integral de debug/perfis; comparar timestamp do diagnóstico global.
- [ ] Não assumir que hit de cache comprova nova execução do CLI; conferir representação, perfis e limitações operacionais.

Veja [fluxos](flows.md), [parâmetros](slice-parameters.md) e [limitações](operations.md).
