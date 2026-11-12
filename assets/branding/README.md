# Brand artwork

| Asset | Use |
| --- | --- |
| [social-preview.svg](social-preview.svg) | Editable 1280 × 640 social card |
| [social-preview.png](social-preview.png) | Upload-ready GitHub repository social preview |
| [app-icon.svg](app-icon.svg) | Scalable desktop and app-store icon with transparent margins |
| [app-icon.icns](app-icon.icns) | macOS app/Dock icon, including 16–1024 px representations |
| [favicon.svg](favicon.svg) | Simplified browser-tab mark designed for 16 px |

Upload the PNG under **Settings → Social preview** on GitHub. To export it again from the SVG, run the following from this directory. [GitHub social preview requirements](https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/customizing-your-repositorys-social-media-preview)

```sh
rsvg-convert --format png --output social-preview.png social-preview.svg
```

Install the desktop SVG at `/app/share/icons/hicolor/scalable/apps/<APP_ID>.svg` inside the Flatpak, using the same app ID as the desktop file's `Icon` value. [Flatpak icon conventions](https://docs.flatpak.org/en/latest/conventions.html#application-icons)

The native macOS build copies `app-icon.icns` to `Contents/Resources/AppIcon.icns` and references it with `CFBundleIconFile` in `Contents/Info.plist`. [Apple's bundle icon reference](https://developer.apple.com/library/archive/documentation/General/Reference/InfoPlistKeyReference/Articles/CoreFoundationKeys.html#//apple_ref/doc/uid/TP40009249-SW10). Regenerate it from the SVG on macOS with `sh tools/build-icons.sh` (requires `rsvg-convert` and Apple's `iconutil`), then rebuild with `sh tools/build-native.sh`. Ordinary native builds use the checked-in ICNS and need no image-conversion tools.

Serve the favicon as `image/svg+xml` and reference it from the document head.

```html
<link rel="icon" href="/favicon.svg" type="image/svg+xml" sizes="any">
```

The SVGs use the game's [cheerful rail palette](../../data/ui/theme.v1.json). The card title uses outlined [Manrope](https://github.com/googlefonts/manrope) lettering, so it renders without a font install. The globe is based on the project's [Natural Earth snapshot](../../data/prototypes/natural-earth-110m.metadata.json). Its route curves are decorative.
