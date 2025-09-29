FROM golang:1.25-bookworm AS builder

ENV GOOS=linux
ENV GOARCH=amd64
ENV GOCACHE=/root/.cache/go-build
ENV GOMODCACHE=/go/pkg/mod

# Install magicpak
ADD https://github.com/coord-e/magicpak/releases/download/v1.4.0/magicpak-x86_64-unknown-linux-musl /usr/bin/magicpak
RUN chmod +x /usr/bin/magicpak

# Install libvips build dependencies and build libvips from source
RUN apt-get update && apt-get install -y \
  build-essential \
  pkg-config \
  libglib2.0-dev \
  libexpat1-dev \
  libtiff5-dev \
  libspng-dev \
  libjpeg62-turbo-dev \
  meson \
  ninja-build \
  wget && \
  rm -rf /var/lib/apt/lists/*

WORKDIR /tmp
ARG LIBVIPS_VERSION="8.17.2"
RUN wget "https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.xz" && \
  tar xf "vips-${LIBVIPS_VERSION}.tar.xz" && \
  cd "vips-${LIBVIPS_VERSION}" && \
  meson setup build --prefix / && \
  cd build && \
  meson compile && \
  meson install

# Build the Go application
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=${GOCACHE} \
  --mount=type=cache,target=${GOMODCACHE} \
  go mod download

COPY . .
RUN --mount=type=cache,target=${GOCACHE} \
  --mount=type=cache,target=${GOMODCACHE} \
  go build -ldflags '-s -w' -trimpath -o /app/main

RUN /usr/bin/magicpak -v /app/main /bundle

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /usr/bin/vips /usr/bin/vips
COPY --from=builder /bundle /

CMD ["/app/main"]
