ARG ORCA_VERSION=2.5.0

FROM golang:1.23-bookworm AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/orca-slicer-api ./cmd/server

# A partir da 2.5.0 o AppImage nao embarca mais os JSON dos perfis: em
# resources/profiles ficam apenas caches binarios .opc por vendor (formato
# cereal interno do Slic3r, sem documentacao publica utilizavel). Sem os JSON a
# API nao consegue resolver a cadeia de "inherits" dos perfis do usuario.
#
# Este stage busca os JSON no repositorio do Orca na MESMA tag do AppImage e
# guarda apenas os arquivos .json em /out/profiles-json.
FROM golang:1.23-bookworm AS vendorprofiles

ARG ORCA_VERSION

RUN set -eux; \
    git clone --depth 1 --branch "v${ORCA_VERSION}" --filter=blob:none --no-checkout \
        https://github.com/OrcaSlicer/OrcaSlicer.git /src; \
    cd /src; \
    git sparse-checkout set resources/profiles; \
    git checkout "v${ORCA_VERSION}"; \
    test -d resources/profiles; \
    test -n "$(find resources/profiles -name '*.json' -print -quit)"; \
    mkdir -p /out/profiles-json; \
    find resources/profiles -name '*.json' -print0 \
        | tar --null -cf /tmp/profiles.tar --files-from -; \
    tar -xf /tmp/profiles.tar -C /out/profiles-json; \
    rm -f /tmp/profiles.tar; \
    test -n "$(find /out/profiles-json -name '*.json' -print -quit)"; \
    find /out/profiles-json -name '*.json' | wc -l

FROM ubuntu:24.04 AS orca

ARG ORCA_VERSION

RUN apt-get update \
	&& apt-get install -y --no-install-recommends ca-certificates curl fuse file libopengl0 libglu1-mesa libgl1 libglx0 libegl1 libglvnd0 libsm6 libxext6 libxrender1 \
	&& update-ca-certificates \
	&& curl -L -o /tmp/orca.AppImage "https://github.com/SoftFever/OrcaSlicer/releases/download/v${ORCA_VERSION}/OrcaSlicer_Linux_AppImage_Ubuntu2404_V${ORCA_VERSION}.AppImage" \
	&& chmod +x /tmp/orca.AppImage \
	&& cd /tmp \
	&& /tmp/orca.AppImage --appimage-extract \
	&& rm /tmp/orca.AppImage \
	&& rm -rf /var/lib/apt/lists/*

FROM ubuntu:24.04

ARG ORCA_VERSION

LABEL org.opencontainers.image.title="Orca Slicer API"
LABEL org.opencontainers.image.description="REST API em Go para controlar OrcaSlicer CLI"
LABEL org.opencontainers.image.source="https://github.com/Brook-sys/orca-slicer-api"
LABEL org.opencontainers.image.vendor="Brook-sys"
LABEL org.opencontainers.image.orca.version="${ORCA_VERSION}"

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
	ca-certificates curl \
	libopengl0 libglu1-mesa \
	libgl1 libgl1-mesa-dri libglx0 libegl1 libglvnd0 \
	libsm6 libxext6 libxrender1 \
	libgtk-3-0 \
	libgstreamer1.0-0 libgstreamer-plugins-base1.0-0 \
	libwebkit2gtk-4.1-0 \
	&& update-ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /out/orca-slicer-api /app/orca-slicer-api
COPY --from=vendorprofiles /out/profiles-json /app/profiles-json
COPY --from=orca /tmp/squashfs-root /app/squashfs-root

ENV PORT=3000
ENV DATA_PATH=/app/data
ENV ORCASLICER_PATH=/app/squashfs-root/AppRun
# Perfis vendor em JSON. O AppImage 2.5.0 so traz .opc nesta arvore; os JSON
# vem do stage vendorprofiles, na mesma tag.
ENV ORCA_PROFILES_PATH=/app/profiles-json/resources/profiles
ENV ORCA_VERSION=${ORCA_VERSION}

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
	CMD curl -f http://localhost:3000/health || exit 1

CMD ["/app/orca-slicer-api"]