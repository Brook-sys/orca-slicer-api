# Erros e diagnóstico

Erros dos handlers usam `Content-Type: application/json`:

```json
{"message":"Slicer is busy"}
```

O campo não é `error`. Respostas de roteamento do Go (404/405 etc.) podem ser texto, e downloads interrompidos podem não permitir substituir a resposta por JSON.

| HTTP | Situação |
|---|---|
| 400 | Multipart/JSON inválido, arquivo ausente/tipo incorreto, limite por arquivo, categoria/nome inválidos, pai ausente/ciclo ou exportType inválido |
| 404 | Perfil/job/alias não encontrado; listagem built-in sem diretório suportado |
| 409 | `Slicer is busy`, ou import sem overwrite para perfil existente |
| 408 | `Slicing cancelled or timed out` durante execução CLI |
| 500 | CLI, filesystem, download externo ou thumbnail falhou |
| 503 | Health essencial falhou |

## Perfis 2.5.0

`parent profile was not found`: conferir raiz `ORCA_PROFILES_PATH=/app/profiles-json/resources/profiles`, disponibilidade do pai e aliases. Ligar `resolveProfiles` não corrige uma raiz errada: nomes já são sempre resolvidos.

`The input preset file is invalid and can not be parsed`: inspecionar perfis finais e saída do CLI. Para JSON multipart, envie perfis planos; esse caminho não resolve herança. Sanitização só ajusta `from` e remove um campo conhecido do processo; não é correção universal.

Warnings de fallback de nomes exigem revisão. Fallback por tokens ambíguo é rejeitado, não escolhe arbitrariamente um candidato. Configure um alias explícito ou use nome inequívoco após conferir os perfis.

## Posicionamento

Foi observado código CLI `-102` com um cubo na origem fora da área imprimível usando `arrange=false`; o mesmo caso passa com `arrange=true`. Um STL de 5.952 triângulos também passou com `arrange=false`. O default continua **false** e não foi demonstrada regressão geral nele. Revise escala, orientação, coordenadas e dimensões da mesa; arrange não corrige todo erro geométrico.

## Sequência de diagnóstico

1. Registrar HTTP e `message` sem confundir corpo JSON de erro com G-code.
2. Conferir `/health`, versão efetiva e raiz JSON.
3. Inspecionar `/profiles/resolve`: resultado e warnings.
4. Consultar `/slice/debug`: `output`, `slicerError`, `resultJson.error_string` e perfis finais.
5. Reproduzir com modelo sintético e cache de teste controlado. Não publicar logs ou perfis pessoais.
