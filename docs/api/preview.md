# Preview

```
POST /slice/preview
```

Endpoint projetado para dashboards que precisam de uma pré-visualização rápida.

## Comportamento

- Força `enableSupport=true` internamente (ignora valor enviado)
- Gera G-code completo
- Detecta uso de suporte procurando por `;TYPE:SUPPORT`
- Retorna **JSON** (não o G-code binário)
- Sempre gera thumbnail PNG 160x160

## Resposta

```json
{
  "usesSupport": true,
  "printTime": 12450.3,
  "filamentUsedG": 18.7,
  "filamentUsedMm": 12450.3,
  "thumbnail": "iVBORw0KGgoAAAANSUhEUgAAAKAAAACgCAYAAACLz2ct..."
}
```

## Campos

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `usesSupport` | bool | `true` se encontrou camadas de suporte |
| `printTime` | number | Tempo estimado em segundos |
| `filamentUsedG` | number | Filamento em gramas |
| `filamentUsedMm` | number | Filamento em milímetros |
| `thumbnail` | string | PNG 160x160 em base64 |

## Regras

- `thumbnail` é sempre retornado (mesmo se `generateImage=false`)
- Arquivos temporários são limpos automaticamente
- `thumbnail` pode vir vazio se falhar a extração

## Exemplo

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true
```
