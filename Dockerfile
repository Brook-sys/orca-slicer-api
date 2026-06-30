FROM golang:1.23-bookworm AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/orca-slicer-api ./cmd/server

FROM ubuntu:24.04

RUN apt-get update \
	&& apt-get install -y --no-install-recommends \
	ca-certificates curl \
	libgl1 libgl1-mesa-dri libegl1 \
	libgtk-3-0 \
	libgstreamer1.0-0 libgstreamer-plugins-base1.0-0 \
	libwebkit2gtk-4.1-0 \
	&& update-ca-certificates \
	&& rm -rf /var/lib/apt/lists/*

COPY --from=build /out/orca-slicer-api /app/orca-slicer-api

ENV PORT=3000
ENV DATA_PATH=/app/data
ENV ORCASLICER_PATH=/app/squashfs-root/AppRun

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
	CMD curl -f http://localhost:3000/health || exit 1

CMD ["/app/orca-slicer-api"]
