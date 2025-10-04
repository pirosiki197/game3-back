#!/bin/bash

set -euo pipefail

LIBVIPS_VERSION="8.17.2"
INSTALL_PREFIX="${1:-/}"

cd /tmp

wget "https://github.com/libvips/libvips/releases/download/v${LIBVIPS_VERSION}/vips-${LIBVIPS_VERSION}.tar.xz"
tar xf "vips-${LIBVIPS_VERSION}.tar.xz"
cd "vips-${LIBVIPS_VERSION}"

meson setup build --prefix "${INSTALL_PREFIX}"
cd build
meson compile
meson install

rm -rf /tmp/vips-*
