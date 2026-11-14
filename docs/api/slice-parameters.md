# Parâmetros de slicing

Multipart para `/slice` e `/slice/async`; `/slice/preview` aceita apenas um modelo no campo `file`.

| Campo | Default/comportamento |
|---|---|
| `file` | Modelo único; STL, STEP/STP, 3MF (extensão sem distinção de caixa) |
| `files` repetido | Vários modelos no sync/async; se presente, tem prioridade sobre `file` |
| `printer`, `preset`, `filament` | Nomes salvos ou vendor; sempre resolvidos/achatados pela API |
| `printerProfile`, `presetProfile` | Arquivos JSON diretos; usa o primeiro de cada campo |
| `filamentProfile` repetido | Um ou mais JSON diretos de filamento |
| `overrides` | String JSON com objetos `printer`, `preset`, `filament`; default `{}` |
| `resolveProfiles` | Flag legada, não desativa resolução de nomes |
| `sanitizeProfiles` | Default `SANITIZE_PROFILES` ou false; define `from=system`, remove `small_perimeter_speed` dos presets |
| `generateImage` | Default `GENERATE_IMAGE` ou false; thumbnail no G-code, não no export 3MF |
| `exportType` | Vazio ou `gcode` para G-code; `3mf` para 3MF; demais valores retornam 400 |
| `plate` | `1`; encaminhada ao CLI, sem validação numérica própria |
| `arrange`, `orient` | false; encaminha `--arrange` e `--orient` como 0/1 |
| `bedType` | Opcional; encaminha `--curr-bed-type` |
| `multicolorOnePlate` | false; quando true, passa `--allow-multicolor-oneplate` |
| `enableSupport` | Ausente preserva preset; presente injeta bool `enable_support` |
| `brimType` | Ausente preserva preset; true → `auto_brim`, false → `no_brim` |
| `printSequenceByObject` | false; sempre define preset como `by layer`; true define `by object` |

Boolean true: exatamente `true`, `1`, `on`. Qualquer outro texto é false. Para flags com default global, formulário e ambiente são combinados por OR: `false` não desliga ambiente true.

## Precedência

- Perfil multipart substitui o nome da mesma categoria. Não resolve herança: envie perfil completo e plano.
- Overrides são aplicados ao perfil, depois sanitização e controles de preset. `enableSupport`, `brimType` e sequência podem sobrescrever overrides.
- Apenas quando existem **printer e preset** a API passa `--load-settings` com ambos. Forneça o par para modelos sem configuração própria. Sem perfis, o sucesso depende do que o modelo/CLI já contém.
- Nenhuma alteração temporária é salva no perfil original.

```bash
-F 'overrides={"preset":{"layer_height":"0.2"},"filament":{"filament_flow_ratio":["1"]}}'
```

## Limites

Cada modelo e cada perfil direto multipart é lido com teto de 100.000.000 bytes. O teto de 4.000.000 bytes se aplica ao CRUD/import persistente, não aos perfis diretos do slice. `ParseMultipartForm` usa limiar de memória, **não limite global do corpo**; não há teto agregado de arquivos. A extensão aceita não prova que a geometria ou o JSON será processável.

No preview, suporte e geração de imagem são forçados a true. Use export G-code no preview; `3mf` não é um caminho de extração de thumbnail/suporte validado.
