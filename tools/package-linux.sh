#!/bin/sh
# Build the Linux release packages: dist/wwhsrt_<version>_amd64.deb and dist/wwhsrt-<version>-x86_64.flatpak.
# Usage: sh tools/package-linux.sh <version> [deb|flatpak|stage]...   (version without the leading v, e.g. 0.1.0)
# Needs a Linux host with dpkg-deb for the .deb and flatpak-builder for the Flatpak; `stage` alone runs anywhere.
set -eu
cd "$(dirname "$0")/.."
version=${1:?usage: sh tools/package-linux.sh <version> [deb|flatpak|stage]...}
shift
echo "$version" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$' ||
  { echo "version must look like 1.2.3 or 1.2.3-rc.1, got $version" >&2; exit 2; }
[ $# -gt 0 ] || set -- deb flatpak
id=io.github.adsouza.wwhsrt
out=build/linux
stage=$out/stage
rm -rf "$stage" && mkdir -p "$stage/bin" dist

# Ebitengine loads X11 and OpenGL with dlopen, so the binary needs no cgo and cross-compiles from any host.
ldflags="-s -w -X main.version=$version"
if [ "${WWHSRT_UNCAPPED:-}" = 1 ]; then
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$ldflags" -o "$stage/bin/wwhsrt" ./cmd/wwhsrt
else
  GOMEMLIMIT=768MiB GOMAXPROCS=2 CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -p 1 -trimpath -ldflags "$ldflags" -o "$stage/bin/wwhsrt" ./cmd/wwhsrt
fi

# One install tree, laid out relative to a prefix: /usr for the .deb, /app for the Flatpak.
install -d "$stage/share/applications" "$stage/share/metainfo" "$stage/share/icons/hicolor/scalable/apps" \
  "$stage/share/doc/wwhsrt"
install -m 644 packaging/linux/$id.desktop "$stage/share/applications/$id.desktop"
sed -e "s/@VERSION@/$version/" -e "s/@DATE@/$(date -u +%Y-%m-%d)/" packaging/linux/$id.metainfo.xml \
  > "$stage/share/metainfo/$id.metainfo.xml"
install -m 644 assets/branding/app-icon.svg "$stage/share/icons/hicolor/scalable/apps/$id.svg"
{
  echo "World-wide High-speed Rail Tycoon $version"
  echo "Source: https://github.com/adsouza/wwhsrt"
  echo "License: AGPL-3.0 (full text below). Map, terrain and network data credits follow it."
  echo
  cat LICENSE
  echo
  cat assets/terrain/ATTRIBUTION.md assets/scenario/NETWORK_ATTRIBUTION.md
} > "$stage/share/doc/wwhsrt/copyright"
chmod 644 "$stage/share/doc/wwhsrt/copyright"

for target in "$@"; do
  case $target in
    stage) echo "Staged $stage" ;;
    deb)
      root=$out/deb
      rm -rf "$root" && mkdir -p "$root/DEBIAN" "$root/usr"
      cp -R "$stage/." "$root/usr/"
      # Debian sorts ~ before everything, so 1.0.0~rc.1 installs before 1.0.0; semver uses - for the same idea.
      debversion=$(echo "$version" | tr - '~')
      size=$(du -sk "$root/usr" | cut -f1)
      cat > "$root/DEBIAN/control" <<EOF
Package: wwhsrt
Version: $debversion
Architecture: amd64
Maintainer: Antonio D'souza <275832+adsouza@users.noreply.github.com>
Installed-Size: $size
Depends: libc6, libx11-6, libxrandr2, libxinerama1, libxi6, libxrender1, libxcursor1, libxext6, libgl1
Recommends: libgl1-mesa-dri, xclip | xsel
Section: games
Priority: optional
Homepage: https://github.com/adsouza/wwhsrt
Description: strategic high-speed railway business game
 Operate and expand high-speed rail networks across real geography: plan
 monthly business turns, watch an animated operating day, and compete for
 passengers through fares, frequency, speed and useful connections.
 .
 Single-player runs offline. Saves live under ~/.config/wwhsrt.
EOF
      dpkg-deb --build --root-owner-group "$root" "dist/wwhsrt_${debversion}_amd64.deb"
      ;;
    flatpak)
      # The manifest installs the staged tree; flatpak-builder only adds the sandbox and AppStream checks.
      flatpak-builder --user --force-clean --install-deps-from=flathub --repo="$out/flatpak-repo" \
        "$out/flatpak-build" packaging/linux/$id.yml
      flatpak build-bundle --runtime-repo=https://dl.flathub.org/repo/flathub.flatpakrepo \
        "$out/flatpak-repo" "dist/wwhsrt-$version-x86_64.flatpak" $id
      ;;
    *) echo "unknown target $target" >&2; exit 2 ;;
  esac
done
ls -l dist
