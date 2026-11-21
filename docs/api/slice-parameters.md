# Parâmetros de slicing

Multipart para `/slice`, `/slice/async` e `/slice/preview`. Os três endpoints aceitam exatamente um STL e o mesmo contrato de quantidade/mesa.

| Campo | Default/comportamento |
|---|---|
| `file` | Um único STL; extensão sem distinção de caixa |
| `files` | Alias legado para um único STL; não combine com `file` nem repita arquivos |
| `quantity` | 1; inteiro de 1 a 512, incluindo o original. Acima de 1 exige `arrange=true`; nunca é reduzido automaticamente |
| `plate` | `1`; qualquer outro valor retorna 400 |
| `arrange` | false; obrigatório para `quantity>1`. O Orca decide posição e espaçamento |
| `arrangeSpacing` | 0 = espaçamento automático. Negativo/não finito ou positivo retorna 400; o CLI 2.5.0 não oferece caminho suportado para espaçamento customizado |
| `orient` | false; encaminha `--orient 0/1`. A presença de `rotate`, `rotateX` ou `rotateY`, mesmo com valor `0`, é incompatível com `orient=true` |
| `rotate`, `rotateX`, `rotateY` | Ausentes; graus finitos em Z/X/Y. Exigem `orient=false`, inclusive quando valem `0`. A simples presença de qualquer um aciona `--ensure-on-bed` para repor no Z da mesa o volume que o Orca 2.5.0 centraliza na origem |
| `printer`, `preset`, `filament` | Nomes salvos ou vendor; sempre resolvidos/achatados pela API |
| `printerProfile`, `presetProfile` | Arquivos JSON diretos; usa o primeiro de cada campo |
| `filamentProfile` repetido | Um ou mais JSON diretos de filamento |
| `overrides` | String JSON com objetos `printer`, `preset`, `filament`; default `{}` |
| `resolveProfiles` | Flag legada, não desativa resolução de nomes |
| `sanitizeProfiles` | Default `SANITIZE_PROFILES` ou false; define `from=system`, remove `small_perimeter_speed` dos presets |
| `generateImage` | Default `GENERATE_IMAGE` ou false; injeta thumbnail no G-code sync/async. O preview gera a imagem separadamente e força esta opção a false internamente |
| `exportType` | Vazio ou `gcode`; qualquer outro valor, inclusive `3mf`, retorna 400. ZIP não é suportado |
| `bedType` | Opcional; encaminha `--curr-bed-type` |
| `multicolorOnePlate` | false; quando true, passa `--allow-multicolor-oneplate` |
| `enableSupport` | Ausente preserva preset; presente escreve `enable_support` como **string `"1"`/`"0"`**. Boolean JSON é ignorado pelo Orca 2.5.0 nesse caminho |
| `brimType` | Ausente preserva preset; true → `auto_brim`, false → `no_brim` |
| `printSequenceByObject` | false; sempre define preset como `by layer`; true define `by object` |

Boolean true no multipart: exatamente `true`, `1`, `on`. Qualquer outro texto é false. Para flags com default global, formulário e ambiente são combinados por OR: `false` não desliga ambiente true.

## Precedência

- Perfil multipart substitui o nome da mesma categoria. Não resolve herança: envie perfil completo e plano.
- Overrides são aplicados ao perfil, depois sanitização e controles de preset. `enableSupport`, `brimType` e sequência podem sobrescrever overrides.
- Apenas quando existem **printer e preset** a API passa `--load-settings` com ambos. Forneça o par para modelos sem configuração própria. Sem perfis, o sucesso depende do que o CLI já consegue usar.
- Nenhuma alteração temporária é salva no perfil original.
- Transformações explícitas são passadas antes de `arrange`/`orient`; o Orca continua autoritativo para a geometria final.

```bash
-F 'overrides={"preset":{"layer_height":"0.2"},"filament":{"filament_flow_ratio":["1"]}}'
```

## Limites e rejeições

Cada modelo e cada perfil direto multipart é lido com teto de 100.000.000 bytes. O teto de 4.000.000 bytes se aplica ao CRUD/import persistente, não aos perfis diretos do slice. `ParseMultipartForm` usa limiar de memória, **não limite global do corpo**; não há teto agregado de arquivos. A extensão aceita não prova que a geometria ou o JSON será processável.

Retornam 400 antes do fatiamento, entre outros casos: múltiplos STLs, STEP/3MF, `quantity=0` ou maior que 512, `quantity>1` sem arrange, `plate!=1`, export não G-code, espaçamento positivo e `orient=true` com qualquer campo de rotação presente, mesmo valendo `0`. Depois de o CLI concluir com sucesso, quantidade ou mesas divergentes retornam 422 no sync/preview; falha do próprio CLI retorna 500. No async, ambos aparecem como job `failed` após o 202. Nenhum resultado parcial é entregue. Consulte [slicing](slicing.md) e [erros](errors.md).
