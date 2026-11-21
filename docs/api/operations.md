# Operação, segurança e limitações

## Exposição de rede

A API **não autentica**, não autoriza por usuário e não implementa TLS. Escuta `:3000` por padrão. Para instalações novas, prefira bind local/rede confiável ou proxy com autenticação, TLS, limites de corpo e rate limiting.

CORS é apenas política de navegador, com default `*`. Não protege endpoints de outros clientes. Todos os clientes com acesso podem importar/remover perfis, consumir CPU, consultar jobs e ler diagnósticos.

- `/health` expõe caminhos; `/slice/debug` inclui perfis, argumentos, saída CLI e caminhos temporários. Trate respostas/logs como potencialmente sensíveis.
- Import HTTPS inicial não é proteção SSRF completa: não há allowlist de host, bloqueio de endereços internos nem política restritiva de redirects. Restrinja egress e acesso à rota.
- JSON válido não significa perfil seguro: configurações podem carregar comandos G-code inadequados. Use apenas perfis/fontes confiáveis e revise saída antes de imprimir.
- Extensões e limites por arquivo não limitam corpo agregado, uso total de RAM/disco ou complexidade geométrica. Configure limites externos e monitore processo e CLI separadamente.
- O servidor usa `http.ListenAndServe` sem timeouts HTTP explícitos. O timeout do CLI não limita uploads lentos.

## Cache e revalidação

O servidor configura TTL de **1 hora** e máximo de **100 entradas**, com cópias em `DATA_PATH/cache`. O índice é somente em memória, não restaurado do disco. Limpeza/evicção de arquivos ocorre periodicamente; não há quota de bytes do cache.

A chave inclui o STL (bytes e nome), todos os settings, overrides, perfis multipart, conteúdo efetivo dos perfis resolvidos (incluindo herança) e SHA-256 do executável. Os perfis são resolvidos antes de cada lookup. Falha de serialização ou leitura do executável desativa o cache para aquela chamada. Preview e slice podem usar representações/settings diferentes e, portanto, chaves distintas.

Snapshots privados são publicados somente após cópia completa, em diretórios únicos. Resultados validados incluem no snapshot o G-code, `result.json` e `plate_1.statistics.json`. **Todo hit repassa pela validação do contrato de arquivo, mesa, quantidade e identidades.** Snapshot sem evidência, com evidência inválida ou que falhe nessas validações é um miss e provoca novo fatiamento, nunca resposta parcial. Não há hash do snapshot nem validação semântica do G-code: alterações que preservem um `plate_1.gcode` não vazio e as evidências válidas podem passar.

Cada hit recebe outra cópia independente, pertencente ao consumidor: remoção pelo handler, substituição, expiração e evicção não removem arquivos retornados a outros consumidores ou jobs. Falha ao copiar não transfere ownership do workdir de origem. A limpeza de órfãos remove somente diretórios `snapshot-*` inativos; diretórios legados podem exigir limpeza manual em ambiente isolado.

Limites residuais: hash do executável e cópias acrescentam I/O por chamada; não há métricas de custo/hit. Bibliotecas externas, recursos implícitos do CLI, ambiente e alterações concorrentes de arquivos durante resolução/execução não são capturados como snapshot atômico. O índice não sobrevive a restart. Cópias entregues a jobs continuam sujeitas ao lifecycle de jobs, não ao TTL do cache; não há garantia de coleta completa de temporários.

Em validações que mudem parâmetros, não aceite hit de cache como evidência de novo fatiamento. Em ambiente de teste isolado e sem processos ativos, reinicie o serviço e limpe somente seu diretório de cache de teste. Não elimine dados compartilhados sem autorização.

## Persistência e recursos

- Backup principal: perfis e `.source.json`, aliases e configurações locais. Proteja URLs de fonte que possam conter dados sensíveis.
- Jobs persistem estado, não uma entrega durável dos arquivos. Após restart não conte com download de job antigo.
- Retenção de registros de jobs é 24 horas, verificada a cada hora. Não há quota de disco nem coleta completa de temporários em todos os caminhos de erro.
- Lock vale por processo, não entre réplicas. Não compartilhe `DATA_PATH` entre servidores esperando isolamento ou escrita coordenada.
- stdout/stderr do CLI retido para debug é limitado a 1 MiB, com marcador de truncamento. Isso não limita o volume produzido pelo processo nem sua CPU/RAM.
- Metadados são lidos incrementalmente e a injeção de thumbnail usa cópia por arquivo temporário. A renderização STL, especialmente ASCII, ainda pode exigir memória proporcional à entrada.
- O parser de preview tem limites próprios e falha fechado acima deles; retenção de 512 camadas/20.000 segmentos produz truncamento explícito, não sucesso parcial da leitura. Consulte [preview](preview.md).
- `/metrics` retorna JSON: `uptime_seconds`, `goroutines`, `memory_alloc_mb` do **Go**, `jobs` e `cache_enabled` quando disponível. Não é formato Prometheus, não mede a memória total do CLI nem confirma integridade do cache.

## Validação local 2.5.0 e gate de release

Nesta revisão, o binário oficial 2.5.0 foi exercitado por HTTP local com STLs sintéticos: sync/async, quantidades exatas, overflow 422/job failed, cache, rotações X/Y/Z, suporte, brim, toolpaths e truncamento. Testes automatizados cobrem evidências ausentes/corrompidas e limites do parser. Isso **não** valida impressão física, equivalência com outra versão ou carga de produção. Imagem de contêiner, publicação e deploy devem ser verificados separadamente.

Antes de uma release ou atualização autorizada, em volume/porta isolados:

- [ ] Rodar formatação, `go test ./...`, `go test -race ./...`, `go vet ./...` e build na revisão final.
- [ ] Conferir `ORCA_VERSION` no Dockerfile/workflow e construir a imagem amd64 da revisão alvo.
- [ ] Confirmar o banner do executável e a raiz JSON efetiva, incluindo pais necessários.
- [ ] Repetir na imagem final os fluxos HTTP estritos, perfis, thumbnails, preview, async/cancelamento e 409 concorrente.
- [ ] Exercitar pai ausente, ciclo, JSON inválido, timeout e restart de job.
- [ ] Registrar revisão, comandos, HTTP, duração, tamanho da saída e falhas, sem dados privados.
- [ ] Revisar a imagem efetivamente executada depois de qualquer atualização autorizada. Publicação de tag não comprova redeploy.

## Diagnóstico

1. Consultar `/health`, configuração efetiva e logs.
2. Usar `/profiles/resolve` para pai ausente/warnings; não tentar resolver raiz JSON incorreta mudando uma flag.
3. Consultar `/slice/debug`, principalmente `resultJson.error_string` e `output`.
4. Reproduzir em ambiente isolado com perfis planos e modelo sintético.
5. Documentar limitações encontradas; novas propostas pertencem a [ROADMAP](../ROADMAP.md).
