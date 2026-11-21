# Orca Slicer API

API REST em **Go 1.23, somente biblioteca padrão**, que gerencia perfis JSON e executa o OrcaSlicer CLI. Oferece fatiamento síncrono e assíncrono de um STL por chamada, quantidade exata na mesa 1, prévia com toolpaths limitados, metadados e miniaturas geradas pela API. Não é uma fila distribuída nem um serviço multiusuário isolado.

> **Estado da validação 2.5.0:** o AppImage oficial foi exercitado por HTTP local com STLs sintéticos, cobrindo sync/async, quantidade e overflow, rotações, suporte/brim, preview e truncamento. A validação local não comprova imagem final, publicação ou deploy; cada etapa exige evidência própria. Equivalência com 2.4.0 e impressão física não foram validados. Veja [evidências e limites](docs/api/overview.md).

## Primeiros passos

Para o roteiro completo e conciso, veja [docs/GETTING_STARTED.md](docs/GETTING_STARTED.md).

### 1. Preparar o ambiente

A imagem atual é **somente `linux/amd64`**. Não há suporte ARM64 validado. Instale Docker com Compose e reserve um diretório de dados gravável. A API **não tem autenticação**; use rede confiável ou proxy com controle de acesso. O exemplo limita a publicação ao localhost e mantém a porta padrão 3000.

Para testar a correção no código local, crie `compose.yaml`:

```yaml
services:
  api:
    build:
      context: .
      args:
        ORCA_VERSION: "2.5.0"
    platform: linux/amd64
    ports:
      - "127.0.0.1:3000:3000"
    volumes:
      - ./data:/app/data
    environment:
      ORCA_PROFILES_PATH: /app/profiles-json/resources/profiles
      SLICE_TIMEOUT_SECONDS: "1800"
```

```bash
mkdir -p data
docker compose up -d --build
curl --fail-with-body http://localhost:3000/health
```

O build baixa o AppImage e os JSON upstream da mesma tag; requer acesso à rede. Para usar GHCR em vez do build local, escolha uma tag/digest cuja validação você tenha confirmado. `latest` não prova que a correção está presente.

`/health` confere arquivos e permissões, não executa um slice. `orcaVersion` vem de configuração, não da inspeção do binário. Veja [execução](docs/api/running.md) e [health](docs/api/health.md).

### 2. Cadastrar perfis próprios

Exporte perfis JSON compatíveis com seu equipamento. Os arquivos abaixo são placeholders locais, não assets fornecidos pelo repositório:

```bash
curl --fail-with-body http://localhost:3000/profiles/printers/upload \
  -F file=@printer.json -F name=minha_impressora
curl --fail-with-body http://localhost:3000/profiles/presets/upload \
  -F file=@preset.json -F name=meu_processo
curl --fail-with-body http://localhost:3000/profiles/filaments/upload \
  -F file=@filament.json -F name=meu_filamento
```

`name` é obrigatório e aceita apenas letras ASCII, números, `_` e `-`; o arquivo precisa terminar em `.json`. Upload substitui um perfil de mesmo nome. O JSON é validado sintaticamente, não quanto à segurança ou compatibilidade com a impressora.

### 3. Conferir a resolução e fatiar

```bash
curl --fail-with-body http://localhost:3000/slice/resolve-profiles \
  -H 'Content-Type: application/json' \
  -d '{"printer":"minha_impressora","preset":"meu_processo","filament":"meu_filamento"}'

curl --fail-with-body http://localhost:3000/slice \
  -F file=@modelo.stl \
  -F printer=minha_impressora \
  -F preset=meu_processo \
  -F filament=meu_filamento \
  -D resultado.headers -o resultado.gcode
```

Confira o HTTP, `X-Slice-Quantity`, `X-Slice-Plate: 1`, o G-code e os metadados antes de imprimir. O contrato não retorna ZIP nem resultado parcial; se a quantidade não couber na mesa 1, reduza `quantity` após o erro 422. Não envie um resultado à impressora sem revisar dimensões, temperaturas, movimentos e compatibilidade dos perfis.

## Compatibilidade de perfis no OrcaSlicer 2.5.0

- O AppImage inspecionado contém **6 JSON e 72 caches binários `.opc`**, serializados internamente com cereal. Não são ZIPs nem uma API pública de perfis.
- O repositório upstream na tag `v2.5.0` contém **14.813 JSON** em `resources/profiles`. O stage `vendorprofiles` copia esses JSON para `/app/profiles-json/resources/profiles`, sem substituir os recursos originais do AppImage.
- `ORCA_PROFILES_PATH` deve apontar para essa **raiz**, não `/app/profiles-json` nem uma pasta de categoria.
- Perfis **nomeados** são sempre resolvidos pela API, inclusive com `resolveProfiles=false`: herança recursiva, overrides e remoção de `inherits` antes do CLI. A flag permanece por compatibilidade.
- O leitor de perfis em disco tolera BOM UTF-8. Isso não estende a tolerância aos uploads/imports JSON.
- A busca por tokens equivalentes é último recurso, rejeita ambiguidades e gera `warnings` nos endpoints de resolução quando usada. Não garante equivalência semântica; revise os resultados.
- Perfis multipart são diretos e **não** passam pela resolução de herança. Envie-os completos/planos.

## Capacidades e limites

- `/slice`, `/slice/async` e `/slice/preview` aceitam exatamente um STL. `quantity` vai de 1 a 512, inclui o original e exige `arrange=true` acima de 1. Só há saída `plate_1.gcode`; múltiplos modelos, STEP/3MF, ZIP e mesas adicionais são rejeitados.
- Quando o CLI conclui com sucesso e a validação detecta quantidade ou mesas divergentes, sync/preview retornam 422. Se o próprio CLI falhar, a resposta é 500; no async, ambos aparecem como job `failed` após o 202. Não há G-code parcial. Manifesto/estatísticas e identidades são validados, mas isso não prova ausência de colisões.
- Um único fatiamento por processo, inclusive preview. Chamadas concorrentes recebem `409`; um job async concorrente pode ser aceito e depois falhar. **Não existe fila de espera.**
- Preview executa fatiamento real, preserva suporte solicitado/preset e extrai toolpaths do G-code com limites e truncamento explícito. O thumbnail é separado e não representa o arranjo final.
- Jobs persistem metadados, mas resultados não são recuperados para download após reinício. O download async entrega o único G-code validado.
- A chave de cache inclui settings, modelo, perfis efetivos e executável; cada hit revalida o contrato de arquivo/mesa/quantidade, sem hash do snapshot ou análise semântica do G-code. Recursos implícitos/ambiente do CLI não são capturados, e hit não comprova nova execução.
- Listagem de built-ins não percorre toda a árvore vendor; pode retornar `404` mesmo quando a resolução por nome funciona.
- Limite de modelo é 100.000.000 bytes **por arquivo**. Não há teto global de requisição, quota de disco ou proteção de carga implementados.

## Documentação e desenvolvimento

- [Índice da API](docs/API.md): rotas, parâmetros, exemplos e operação.
- [Segurança e limitações operacionais](docs/api/operations.md).
- [Roadmap](docs/ROADMAP.md): somente planejamento, sem compromisso de entrega.
- [Guia para agentes](AGENTS.md).

```bash
go test ./...
go vet ./...
go build -o /tmp/orca-slicer-api ./cmd/server
```

O servidor pode iniciar sem Orca, mas `/health` fica degradado e slicing não funciona. Para execução local, configure `ORCASLICER_PATH` e `ORCA_PROFILES_PATH` explicitamente; veja [configuração](docs/api/configuration.md).
