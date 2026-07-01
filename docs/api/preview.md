# Preview

## `POST /slice/preview`

Gera uma prévia para dashboard, sem retornar o G-code final.

O endpoint executa slicing real, força suporte ativo e retorna JSON com:

- Se o OrcaSlicer gerou suporte
- Tempo estimado
- Filamento estimado
- Thumbnail PNG base64

## Uso Principal

Use este endpoint antes do `/slice` quando o dashboard precisa decidir se deve avisar o usuário sobre suportes.

## Comportamento

- Entrada: `multipart/form-data`
- Aceita os mesmos campos principais do `/slice`
- Força `enableSupport=true` internamente
- Gera G-code completo em workdir temporário
- Detecta suporte procurando por marcadores no G-code
- Extrai thumbnail PNG 160x160
- Retorna JSON
- Remove arquivos temporários após a resposta

## Multipart Fields

Mesmos campos do `/slice`, com observações:

| Campo | Comportamento no Preview |
|-------|--------------------------|
| `file` | Obrigatório |
| `printer` / `preset` / `filament` | Igual ao `/slice` |
| `printerProfile` / `presetProfile` / `filamentProfile` | Igual ao `/slice` |
| `resolveProfiles` | Recomendado `true` |
| `sanitizeProfiles` | Recomendado `true` para profiles custom |
| `enableSupport` | Ignorado; preview força `true` |
| `generateImage` | Ignorado; preview sempre gera thumbnail |
| `brimType` | Respeitado se enviado |
| `printSequenceByObject` | Respeitado se enviado |

## Detecção de Suporte

A detecção é feita no G-code gerado.

Atualmente considera suporte quando encontra marcadores como:

```txt
;TYPE:SUPPORT
; support
;TYPE:SUPPORT INTERFACE
```

Isso reflete o que o OrcaSlicer realmente gerou, não uma análise geométrica prévia.

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

## Campos da Resposta

| Campo | Tipo | Descrição |
|-------|------|-----------|
| `usesSupport` | bool | `true` se o G-code gerado contém suporte |
| `printTime` | number | Tempo estimado de impressão em segundos |
| `filamentUsedG` | number | Filamento usado em gramas |
| `filamentUsedMm` | number | Filamento usado em milímetros |
| `thumbnail` | string | PNG 160x160 codificado em base64 |

## Exemplo

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@model.stl \
  -F printer=Elegoo_Neptune_4_0_4_nozzle_-OpenNept4une \
  -F preset=0_2mm_standard_0_4_ONP \
  -F filament=PLA_personalizado1_ONP \
  -F resolveProfiles=true \
  -F sanitizeProfiles=true
```

## Erros Comuns

| Status | Causa |
|--------|-------|
| `400` | arquivo ausente ou multipart inválido |
| `409` | slicer ocupado |
| `500` | falha no OrcaSlicer ou geração de imagem |
| `408` | timeout/cancelamento |
