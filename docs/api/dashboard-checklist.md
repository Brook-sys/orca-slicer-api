# Checklist de Integração para Dashboard

## Configuração Inicial

- [ ] Configurar URL base da API
- [ ] Testar `GET /health`
- [ ] Carregar lista de printers: `GET /profiles/printers`
- [ ] Carregar lista de presets: `GET /profiles/presets`
- [ ] Carregar lista de filaments: `GET /profiles/filaments`

## Upload / Import de Profiles

- [ ] Permitir upload JSON por categoria
- [ ] Permitir import por URL raw HTTPS
- [ ] Mostrar `sourceUrl` quando existir
- [ ] Permitir update-from-source
- [ ] Permitir delete

## Preview

- [ ] Enviar `POST /slice/preview` após usuário selecionar modelo e profiles
- [ ] Exibir `thumbnail` base64
- [ ] Exibir tempo estimado (`printTime`)
- [ ] Exibir filamento (`filamentUsedG`, `filamentUsedMm`)
- [ ] Mostrar aviso quando `usesSupport=true`

## Slice Final

- [ ] Enviar `POST /slice`
- [ ] Permitir controlar `enableSupport`
- [ ] Permitir controlar `brimType`
- [ ] Permitir controlar `printSequenceByObject`
- [ ] Ativar `generateImage=true` quando o G-code precisa de thumbnail
- [ ] Baixar arquivo retornado (`.gcode`, `.3mf` ou `.zip`)

## Debug e Status

- [ ] Mostrar status de processamento usando `/slice/status`
- [ ] Em caso de erro, consultar `/slice/debug`
- [ ] Exibir mensagem amigável baseada em `error` ou `resultJSON.error_string`

## Recomendações

- Sempre usar `resolveProfiles=true` para profiles custom com `inherits`
- Sempre usar `sanitizeProfiles=true` para profiles custom vindos do Orca GUI
- Usar `/slice/preview` antes do `/slice` quando quiser detectar suportes
- Usar `GENERATE_IMAGE=true` no container se todas as chamadas devem gerar thumbnail
