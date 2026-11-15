"""Check production import boundaries without initializing graphics or audio."""

import json
import subprocess

decoder = json.JSONDecoder()
raw = subprocess.check_output(
    ["go", "list", "-json", "./internal/domain", "./pkg/gameapi", "./internal/application", "./pkg/terrain"], text=True
)
packages = []
while raw.strip():
    package, end = decoder.raw_decode(raw.lstrip())
    packages.append(package)
    raw = raw.lstrip()[end:]
for package in packages:
    name = package["ImportPath"]
    for imported in package.get("Imports", []):
        assert "ebiten" not in imported and not imported.startswith("wwhsrt/pkg/app"), (name, imported)
        if name.endswith("/domain"):
            assert imported not in ["os", "time", "net", "log", "log/slog", "math/rand"], (name, imported)
            assert not imported.startswith("wwhsrt/") or imported == "wwhsrt/pkg/gameapi", (name, imported)
        if name == "wwhsrt/pkg/gameapi":
            assert not imported.startswith("wwhsrt/"), (name, imported)
deps = subprocess.check_output(["go", "list", "-deps", "./cmd/headless"], text=True)
assert "ebiten" not in deps and "wwhsrt/pkg/render" not in deps
print("Domain/API/application/terrain boundaries and graphics-free headless composition passed")
