# Checklist de cliente/dashboard

Esta lista orienta uma integração externa; não descreve um frontend fornecido pelo projeto.

- [ ] Configurar URL e acesso por rede confiável/proxy autenticado. CORS não autentica.
- [ ] Conferir health sem confundi-lo com teste de slice.
- [ ] Carregar listas de perfis salvos, com source/checksum; não depender do catálogo built-in, que retorna 404 no layout vendor normal.
- [ ] Exigir `name` ASCII válido no upload; avisar que upload sobrescreve.
- [ ] Oferecer import/update apenas a operadores confiáveis; há risco SSRF.
- [ ] Resolver perfis nomeados e exibir warnings; não oferecer `resolveProfiles=false` como bypass de herança.
- [ ] Exigir perfis completos/planos nos uploads diretos de slice.
- [ ] Diferenciar download de arquivo/ZIP e corpo JSON de erro (`message`).
- [ ] Exibir unidades: segundos, gramas e milímetros; estimativas não são medições reais.
- [ ] Avisar que preview roda CLI, força suporte/imagem e não garante necessidade física de suporte.
- [ ] Não prometer imagem de STEP/3MF ou de toda a mesa com múltiplos modelos.
- [ ] Tratar 409 e async failed por ocupado; não mostrar fila inexistente.
- [ ] Usar job específico para polling; guardar download antes de reiniciar servidor.
- [ ] Apresentar cancelamento como tentativa e conferir estado depois.
- [ ] Não publicar conteúdo integral de debug/perfis; comparar timestamp do diagnóstico global.
- [ ] Avisar sobre cache incompleto ao comparar parâmetros; não assumir que toda alteração foi refatiada.

Veja [fluxos](flows.md), [parâmetros](slice-parameters.md) e [limitações](operations.md).
