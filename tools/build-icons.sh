#!/bin/sh
# Developer asset export; normal builds use the checked-in ICNS.
set -eu
cd "$(dirname "$0")/.."
command -v rsvg-convert >/dev/null
command -v iconutil >/dev/null
mkdir -p build
icon_work=$(mktemp -d build/rail-icon.XXXXXX)
trap 'rm -rf "$icon_work"' EXIT HUP INT TERM
icon_set="$icon_work/app-icon.iconset"
mkdir "$icon_set"
for size in 16 32 128 256 512; do
  for scale in 1 2; do
    pixels=$((size * scale))
    suffix=''
    if [ "$scale" = 2 ]; then suffix='@2x'; fi
    rsvg-convert -w "$pixels" -h "$pixels" -o "$icon_set/icon_${size}x${size}${suffix}.png" assets/branding/app-icon.svg
  done
done
iconutil -c icns -o "$icon_work/app-icon.icns" "$icon_set"
mv "$icon_work/app-icon.icns" assets/branding/app-icon.icns
