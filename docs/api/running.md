# Executando

## Requisitos

Go 1.23 para API; OrcaSlicer CLI e bibliotecas nativas para slicing. A imagem usa AppImage **amd64**, portanto somente `linux/amd64` é suportado. Declarar platform não garante viabilidade/performance em host ARM emulado.

## Build local da imagem

```bash
docker build --platform linux/amd64 --build-arg ORCA_VERSION=2.5.0 -t slicer-api:local .
mkdir -p data
docker run --rm --name slicer-api --platform linux/amd64 \
  -p 127.0.0.1:3000:3000 \
  -v "$(pwd)/data:/app/data" \
  -e ORCA_PROFILES_PATH=/app/profiles-json/resources/profiles \
  slicer-api:local
```

Compose equivalente no [README](../../README.md). Porta padrão mantida em 3000; o bind local é recomendação para exemplos, não mudança em deploy existente. Para acesso remoto use rede confiável ou proxy com autenticação/TLS, pois a API não valida credenciais.

O build baixa AppImage e JSON da mesma tag upstream. O stage `vendorprofiles` só copia JSON para a raiz separada `/app/profiles-json/resources/profiles`, preservando o AppImage original. Não basta apontar para `/app/profiles-json`.

## Executar Go fora do contêiner

```bash
export PORT=3000
export DATA_PATH=./data
export ORCASLICER_PATH=/caminho/OrcaSlicer/AppRun
export ORCA_PROFILES_PATH=/caminho/OrcaSlicer-fontes/resources/profiles
export ORCA_VERSION=2.5.0
go run ./cmd/server
```

Os caminhos acima são placeholders. Use JSON da mesma versão do executável. O Go pode iniciar sem Orca, mas health e slicing não ficam funcionais. O servidor local escuta todas as interfaces; limite acesso no ambiente.

## CI, publicação e atualização

`ci.yml` faz build de imagem somente amd64 e `push:false`. **`container.yml` é o único publicador**: usa `ORCA_VERSION=2.5.0`, build amd64, publica em push de main/tag e execução manual; em pull request não publica. Tags incluem branch/tag, `sha-...`, `latest` e variantes Orca conforme regras do workflow.

Não conclua que uma tag publicada foi implantada. Escolha revisão/digest verificável, faça backup dos dados e valide em instância isolada antes de qualquer troca autorizada. Não há deploy concluído registrado nesta correção. Veja [operação](operations.md).
