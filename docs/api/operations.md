# Operação, segurança e limitações

## Exposição de rede

A API **não autentica**, não autoriza por usuário e não implementa TLS. Escuta `:3000` por padrão. A decisão de publicar a porta é do operador; estes documentos não alteram implantação existente. Para instalações novas, prefira bind local/rede confiável ou proxy com autenticação, TLS, limites de corpo e rate limiting.

CORS é apenas política de navegador, com default `*`. Não protege endpoints de outros clientes. Todos os clientes com acesso podem importar/remover perfis, consumir CPU, consultar jobs e ler diagnósticos.

- `/health` expõe caminhos; `/slice/debug` inclui perfis, argumentos, saída CLI e caminhos temporários. Trate respostas/logs como potencialmente sensíveis.
- Import HTTPS inicial não é proteção SSRF completa: não há allowlist de host, bloqueio de endereços internos nem política restritiva de redirects. Restrinja egress e acesso à rota.
- JSON válido não significa perfil seguro: configurações podem carregar comandos G-code inadequados. Use apenas perfis/fontes confiáveis e revise saída antes de imprimir.
- Extensões e limites por arquivo não limitam corpo agregado, quantidade de arquivos, uso de RAM/disco ou complexidade geométrica. Configure limites externos e monitore processo e CLI separadamente.
- O servidor usa `http.ListenAndServe` sem timeouts HTTP explícitos. O timeout do CLI não limita uploads lentos.

## Cache: não tratar como garantia de correção

O servidor configura TTL de **1 hora** e máximo de **100 entradas**, com cópias em `DATA_PATH/cache`. O índice é somente em memória, não restaurado do disco. Limpeza/evicção de arquivos ocorre periodicamente.

A chave inclui modelos ordenados (bytes e nomes), todos os settings, overrides, perfis multipart, conteúdo efetivo dos perfis resolvidos (incluindo herança) e SHA-256 do executável. Os perfis são resolvidos antes de cada lookup. Falha de serialização ou leitura do executável desativa o cache para aquela chamada. O preview preserva o controle de suporte, força apenas G-code e imagem; diferenças de representação podem gerar entradas distintas.

Snapshots privados são publicados somente após cópia completa, em diretórios únicos. Cada hit recebe outra cópia independente, pertencente ao consumidor: remoção pelo handler, substituição, expiração e evicção não removem arquivos retornados a outros consumidores ou jobs. Falha ao copiar não transfere ownership do workdir de origem. A limpeza de órfãos remove somente diretórios `snapshot-*` inativos; diretórios legados podem exigir limpeza manual em ambiente isolado.

Limites residuais: hash do executável e cópias acrescentam I/O por chamada; não há métricas de custo/hit. Bibliotecas externas, recursos implícitos do CLI, ambiente e alterações concorrentes de arquivos durante resolução/execução não são capturados como snapshot atômico. O índice não sobrevive a restart. Cópias entregues a jobs continuam sujeitas ao lifecycle de jobs, não ao TTL do cache; não há garantia de coleta completa de temporários.

Em validações que mudem parâmetros, não aceite hit de cache como evidência de novo fatiamento. Em ambiente de teste isolado e sem processos ativos, reinicie o serviço e limpe somente seu diretório de cache de teste. Não elimine dados compartilhados sem autorização.

## Persistência e recursos

- Backup principal: perfis e `.source.json`, aliases e configurações locais. Proteja URLs de fonte que possam conter dados sensíveis.
- Jobs persistem estado, não uma entrega durável dos arquivos. Após restart não conte com download de job antigo.
- Retenção de registros de jobs é 24 horas, verificada a cada hora. Não há quota de disco nem coleta completa de temporários em todos os caminhos de erro.
- Lock vale por processo, não entre réplicas. Não compartilhe `DATA_PATH` entre servidores esperando isolamento ou escrita coordenada.
- `/metrics` retorna JSON: `uptime_seconds`, `goroutines`, `memory_alloc_mb` do **Go**, `jobs` e `cache_enabled` quando disponível. Não é formato Prometheus, não mede a memória total do CLI nem confirma integridade do cache.

## Gate para atualização 2.5.0

Estado conhecido: AppImage original + perfis achatados pela API fatiou cubo e STL de 5.952 triângulos, este último também com arrange=false. Métricas e diferenças frente à referência 2.4.0 estão em [visão geral](overview.md). Não há equivalência ou impressão física validada. Correção enviada para build; validação completa HTTP e da imagem final continua pendente. Não registrar deploy concluído sem evidência externa.

Em volume/porta de teste isolados e com modelos sintéticos:

- [ ] Rodar `go test ./...`, `go vet ./...` e build.
- [ ] Conferir `ORCA_VERSION` no Dockerfile e workflow; construir imagem amd64 da revisão alvo.
- [ ] Confirmar banner com `AppRun --help`, não somente `orcaVersion` do health.
- [ ] Confirmar `/app/profiles-json/resources/profiles` como raiz JSON, incluindo pais necessários.
- [ ] Resolver cadeias de herança, revisar warnings de tokens e confirmar ausência de `inherits` na saída.
- [ ] Fatiar pela API com `resolveProfiles=true` e false, sem mascarar diferença por cache.
- [ ] Validar saída G-code/3MF, estimativas, overrides, sanitização e thumbnails separadamente.
- [ ] Testar preview, múltiplos modelos, async/poll/download/cancelamento e 409 concorrente.
- [ ] Exercitar pai ausente, ciclo, JSON inválido, timeout e restart de job.
- [ ] Registrar versão/revisão, comando, HTTP, duração, tamanho da saída e falhas, sem dados privados.
- [ ] Revisar imagem efetivamente executada depois de qualquer atualização autorizada. Publicação de tag não comprova redeploy.

## Diagnóstico

1. Consultar `/health`, configuração efetiva e logs.
2. Usar `/profiles/resolve` para pai ausente/warnings; não tentar resolver raiz JSON incorreta mudando uma flag.
3. Consultar `/slice/debug`, principalmente `resultJson.error_string` e `output`.
4. Reproduzir em ambiente isolado com perfis planos e modelo sintético.
5. Documentar limitações encontradas; novas propostas pertencem a [ROADMAP](../ROADMAP.md).
