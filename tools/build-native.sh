#!/bin/sh
set -eu
cd "$(dirname "$0")/.."
mkdir -p build
GOMEMLIMIT=768MiB GOMAXPROCS=2 go build -p 1 -trimpath -o build/wwhsrt ./cmd/wwhsrt
if [ "$(go env GOOS)" = darwin ]; then
  bundle='build/World-wide High-speed Rail Tycoon.app'
  mkdir -p "$bundle/Contents/MacOS" "$bundle/Contents/Resources"
  cp build/wwhsrt "$bundle/Contents/MacOS/wwhsrt.next"
  mv "$bundle/Contents/MacOS/wwhsrt.next" "$bundle/Contents/MacOS/wwhsrt"
  cat assets/terrain/ATTRIBUTION.md assets/scenario/NETWORK_ATTRIBUTION.md > "$bundle/Contents/Resources/TerrainAttribution.txt"
  cp assets/branding/app-icon.icns "$bundle/Contents/Resources/AppIcon.icns.next"
  mv "$bundle/Contents/Resources/AppIcon.icns.next" "$bundle/Contents/Resources/AppIcon.icns"
  cat > "$bundle/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>CFBundleName</key><string>World-wide High-speed Rail Tycoon</string>
  <key>CFBundleIdentifier</key><string>io.github.adsouza.wwhsrt.dev</string>
  <key>CFBundleExecutable</key><string>wwhsrt</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleIconFile</key><string>AppIcon.icns</string>
  <key>CFBundleVersion</key><string>2</string>
  <key>NSHighResolutionCapable</key><true/>
</dict></plist>
PLIST
  touch "$bundle"
fi
