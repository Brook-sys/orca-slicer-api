# Preview

`POST /slice/preview` recebe multipart com **um modelo em `file`** e os parâmetros de perfis/controles descritos em [parâmetros](slice-parameters.md). Executa fatiamento real pelo mesmo lock de sync/async, não apenas renderização.

Força `enableSupport=true` e `generateImage=true`, independentemente dos valores enviados. Requer preset utilizável para que o controle de suporte seja aplicado. Preserve export G-code; o caminho de preview com `exportType=3mf` não é validado para detecção de suporte e extração de imagem.

Resposta ilustrativa:

```json
{
  "usesSupport": false,
  "printTime": 412,
  "filamentUsedG": 1.0,
  "filamentUsedMm": 340.83,
  "thumbnail": "BASE64_DO_PNG"
}
```

Valores são estimativas extraídas da saída, não medições de impressão. `usesSupport` procura marcadores de suporte no G-code gerado; false não prova segurança de impressão sem suportes. A miniatura é gerada pela API, não uma renderização fiel do toolpath. Consulte [thumbnails](neptune4.md).

O preview pode custar um slice completo e não substitui revisão do G-code. Não há garantia de aproveitar o resultado no slice final; o cache atual não considera todos os parâmetros. Pode retornar 400 por entrada, 409 ocupado, 408 timeout/cancelamento e 500 por falha do CLI/imagem.

Exemplos em [examples.md](examples.md).
