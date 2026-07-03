# Orca Slicer API

Uma API REST completa escrita em Go para gerenciar perfis JSON e controlar o **OrcaSlicer CLI** de forma automatizada e escalável via Docker. Criada para integrar fatiamento 3D em sistemas web, Print Farms e CRMs.

---

## 🚀 Recursos Principais

- **Slicing Assíncrono e Síncrono:** Fatie arquivos pequenos na mesma requisição, ou use a fila assíncrona com percentual de progresso em tempo real para arquivos enormes.
- **Plating (Multi-modelo):** Aceita múltiplos arquivos `.stl/.3mf` de uma vez, usando o `--arrange` do Orca para organizá-los na mesa.
- **Gestão Completa de Profiles:** Faça upload, importe via URL, crie aliases, e visualize os embutidos do próprio OrcaSlicer sem encostar em bancos de dados (tudo armazenado em disco de forma portável).
- **Cache Inteligente:** Slices idênticos não gastam CPU duas vezes. Os G-codes e `.3mf` ficam armazenados no cache (`DATA_PATH/cache`) e são servidos imediatamente.
- **Limpeza Automática:** Jobs cancelados/falhados/concluídos antigos e cache expirado são limpos automaticamente de hora em hora.
- **Thumbnails Nativos:** Gera miniaturas em PNG do modelo fatiado no próprio payload G-code.
- **Swagger / OpenAPI:** Documentação da API com interface web de testes embutida.

---

## 🏃 Como rodar (Docker Compose)

A maneira mais rápida de começar é usar o **Docker Compose**. O contêiner já vem com todas as dependências e o motor do OrcaSlicer embutido.

Crie um arquivo `docker-compose.yml`:

```yaml
services:
  slicer-api:
    image: ghcr.io/brook-sys/orca-slicer-api:latest
    ports:
      - "3000:3000"
    volumes:
      - ./data:/app/data
    environment:
      # Gera miniatura PNG de 160x160 junto no G-code por padrão
      - GENERATE_IMAGE=true
      # Tenta herdar de perfis "Base" automaticamente ao invés de falhar
      - RESOLVE_PROFILES=true
      # Sanitiza campos como "from: User" ou velocidades nulas que quebram o CLI
      - SANITIZE_PROFILES=true
      # Tempo máximo que um slicing pode demorar (em segundos)
      - SLICE_TIMEOUT_SECONDS=3600
```

E suba a aplicação:

```bash
docker-compose up -d
```

Verifique se está rodando acessando o status do motor:
```bash
curl http://localhost:3000/health
```

---

## 📚 Como Usar a API: Guia Rápido

Aqui estão os exemplos práticos mais comuns no dia a dia. Todos os arquivos são postados no formato `multipart/form-data`.

### 1. Slicing Síncrono (O básico)
Baixa o G-code final na mesma requisição. Ideal para modelos pequenos ou integrações diretas.

```bash
curl -X POST http://localhost:3000/slice \
  -F file=@meu_arquivo.stl \
  -F printer="Elegoo Neptune 4 0.4 nozzle" \
  -F preset="0.20mm Standard @Elegoo N4 0.4 nozzle" \
  -F filament="Generic PLA @Elegoo" \
  -F generateImage=true \
  -o resultado.gcode
```

### 2. Slicing com Múltiplos Arquivos (Plating)
Você pode enviar várias peças passando a chave `files` múltiplas vezes e pedindo para a API arranjá-las.

```bash
curl -X POST http://localhost:3000/slice \
  -F files=@cabeca.stl \
  -F files=@corpo.stl \
  -F files=@bracos.stl \
  -F arrange=true \
  -o resultado_junto.gcode
```

### 3. Slicing Assíncrono (O melhor para modelos grandes)
Quando o modelo é muito pesado (ex. 100MB), a requisição `POST /slice` convencional pode sofrer _Timeout_ do roteador/proxy. Use a fila Async.

**Passo 3.1:** Inicie o Job:
```bash
curl -X POST http://localhost:3000/slice/async \
  -F file=@dragao.stl
```
A API responde na hora (Código `202 Accepted`):
```json
{
  "id": "e3b0c44298fc1c14",
  "status": "pending",
  "progress": 0,
  "createdAt": "2026-07-03T15:04:05Z"
}
```

**Passo 3.2:** Acompanhe o progresso (Polling):
```bash
curl http://localhost:3000/jobs/e3b0c44298fc1c14
```
_A API retornará `"progress": 45, "status": "slicing"`._

**Passo 3.3:** Faça o Download (quando `"status": "completed"`):
```bash
curl -OJ http://localhost:3000/jobs/e3b0c44298fc1c14/download
```

**Passo Bônus:** Listar ou cancelar jobs:
```bash
# Lista todos os jobs em andamento/concluídos recentemente
curl http://localhost:3000/jobs

# Cancela um job travado/indesejado
curl -X DELETE http://localhost:3000/jobs/e3b0c44298fc1c14
```

### 4. Consultando Profiles disponíveis (Built-ins)
Para saber quais impressoras, filamentos e configs vêm instalados no Orca nativamente, basta pedir a lista:

```bash
curl http://localhost:3000/profiles/builtins/printers
curl http://localhost:3000/profiles/builtins/presets
curl http://localhost:3000/profiles/builtins/filaments
```

### 5. Configurações Dinâmicas Rápidas (Preview & Overrides)
Se você quer simular se o modelo precisa de suportes antes de baixar o G-Code pesado, ou quer injetar um parâmetro na mosca, utilize:

```bash
curl -X POST http://localhost:3000/slice/preview \
  -F file=@modelo.stl \
  -F enableSupport=true \
  -F brimType=false \
  -F printSequenceByObject=true
```

Isso não devolve o arquivo inteiro, apenas um JSON de metadados contendo o tempo de impressão, gramas gastas, thumbnail e se a peça *precisou* de suportes ou não.

---

## 📊 Endpoints Completos

| Categoria | Método & Rota | Funcionalidade |
|-----------|---------------|----------------|
| **Core** | `GET /health` | Retorna saúde da API e caminhos do sistema. |
| **Core** | `GET /metrics` | Estatísticas (uptime, memória e uso da fila/cache). |
| **Slice** | `POST /slice` | Fatiamento Síncrono direto (Retorna G-code/ZIP). |
| **Slice** | `POST /slice/async` | Inicia fatiamento na fila (Retorna Job ID). |
| **Slice** | `POST /slice/preview`| Simula fatiamento e retorna Metadata + Thumbnail PNG 160px. |
| **Jobs** | `GET /jobs` | Lista todos os jobs conhecidos pela API. |
| **Jobs** | `GET /jobs/{id}` | Vê detalhes/percentual de um job. |
| **Jobs** | `GET /jobs/{id}/download` | Baixa o arquivo do job finalizado. |
| **Jobs** | `DELETE /jobs/{id}` | Força o cancelamento daquele fatiamento CLI. |
| **Profiles**| `GET /profiles/builtins/{cat}`| Traz JSON com a lista do que já vem embutido no motor do OrcaSlicer. |
| **Profiles**| `GET /profiles/{category}` | Lista profiles customizados salvos localmente. |
| **Profiles**| `POST /profiles/{category}/upload`| Sobe um novo arquivo JSON permanentemente no servidor. |
| **Docs** | `GET /api-docs` | Swagger UI visual interativo da API. |

---

## 🛠 Desenvolvimento Local & Código-Fonte

Caso deseje clonar o repositório e estender a API:

```bash
# Clone o projeto
git clone https://github.com/Brook-sys/orca-slicer-api.git
cd orca-slicer-api

# Baixe as dependências e rode a suíte de testes
go test ./...

# Rode localmente (é preciso ter o Orca AppRun nativo no sistema)
export ORCASLICER_PATH=/caminho/para/OrcaSlicer/AppRun
export DATA_PATH=./data
export PORT=3000

go run ./cmd/server
```

A arquitetura do código se divide em:
* `cmd/server/main.go` -> Ponto de entrada, injetor de dependências e roteador HTTP (`net/http.ServeMux` nativo do Go 1.22+).
* `internal/slicer/` -> Lógica principal. Controla o binário do Orca, argumentos CLI, Pipes de stderr/stdout, Plating e Cache de G-code.
* `internal/jobs/` -> Gerenciador de estado em memória + persistência em arquivo JSON. Acompanha progresso assíncrono.
* `internal/profiles/` -> Lógica de gerir uploads, aliases e resolução de campos que herdam `(inherits)` de outros arquivos JSON.
* `docs/api/` -> Arquivos mais detalhados sobre as operações e comportamento do motor (Para uso de sysadmins).
