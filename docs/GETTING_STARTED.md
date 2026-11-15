# Primeiros passos

Guia para uma instância de teste isolada. A imagem é somente **linux/amd64** e a API não tem autenticação. O serviço usa porta 3000; limite acesso a localhost/rede confiável ou proxy autenticado. Não é necessário mudar uma implantação existente para seguir o exemplo de teste.

## 1. Preparar e iniciar

Tenha Docker/Compose e um diretório de dados gravável. Use o Compose do [README](../README.md) ou o build/run de [execução](api/running.md). A correção 2.5.0 está em validação: não presuma que uma imagem publicada ou um health 200 comprove o fluxo completo.

Para 2.5.0, confirme `ORCA_PROFILES_PATH=/app/profiles-json/resources/profiles`. Esse é o nível exato dos JSON vendor, não o diretório pai. Confira os demais defaults em [configuração](api/configuration.md).

## 2. Conferir ambiente

```bash
curl --fail-with-body http://localhost:3000/health
```

Revise caminhos, permissões e versão configurada. O endpoint não executa slice e não falha só por raiz JSON indisponível. Veja [health](api/health.md). Confira também o banner do executável conforme o guia de execução.

## 3. Cadastrar três perfis

Forneça seus próprios `printer.json`, `preset.json`, `filament.json`, compatíveis com sua impressora. Use os comandos de upload do [README](../README.md), com nomes fictícios `minha_impressora`, `meu_processo`, `meu_filamento` substituídos conforme necessário.

O nome é obrigatório; upload sobrescreve. JSON válido não significa seguro. Para herança, mantenha os pais disponíveis na raiz vendor/dados. O catálogo built-in não percorre o layout vendor normal e pode retornar 404; isso não impede necessariamente resolver por nome.

## 4. Resolver antes de fatiar

```bash
curl --fail-with-body http://localhost:3000/slice/resolve-profiles \
  -H 'Content-Type: application/json' \
  -d '{"printer":"minha_impressora","preset":"meu_processo","filament":"meu_filamento"}'
```

Confira perfis achatados e warnings. Perfis nomeados são sempre resolvidos, mesmo com `resolveProfiles=false`. Se preferir perfis multipart diretos, envie JSON completo/plano: esse caminho não resolve herança. Veja [resolução](api/resolve-aliases.md).

## 5. Testar um modelo sintético

Execute o exemplo síncrono de [examples.md](api/examples.md), com modelo local simples e perfis selecionados. Confira HTTP, headers, formato de arquivo e saída. Posição/escala precisam caber na mesa; `arrange` é false por padrão e pode ser solicitado quando apropriado.

A chave do cache inclui settings, perfis efetivos e executável; hit não comprova nova execução do CLI. Leia [operação](api/operations.md) antes de validar mudanças. Sucesso de CLI não valida impressão física nem equivalência entre versões.

## 6. Integrar e validar

Depois do sync, teste [preview](api/preview.md) e [jobs async](api/slicing.md), observando limites de concorrência e persistência. Complete o checklist de [operação](api/operations.md) antes de considerar a imagem validada. O estado/evidência atual está em [visão geral](api/overview.md); não há deploy concluído atestado.
