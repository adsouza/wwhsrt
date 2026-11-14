# Reviewing idle lines

Review now frames all authoritative idle warnings, highlights their actual built geometry and names every affected line/fleet in Services. Each named row selects the matching service and focuses its route. On phones, Services exposes the map while idle warnings exist. Infrastructure uses a thick solid warning stroke and endpoint rings; idle leased fleets use a dashed route stroke, a separate legend entry and an **Idle trains** label. This distinction matters when another service is running over the same shared track. Through-fleet warnings cover all route sections. A physical idle warning retains its solid style if an idle fleet also covers it.

`pkg/ui/idle_review.go` maps the existing warning projection to committed alignments and service sections. Native and browser hosts use the shared camera/renderer. No domain rules, financial commands or campaign schema changed. Highlights and names update from current accepted operations, including after reload. Leasing without applying a positive-frequency service retains the warning.

Verification:

- `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/ui ./pkg/app ./pkg/render -count=1`: committed-alignment selection, active/unbuilt exclusions, through-fleet section coverage, native actual-button review, individual selection, framing all geometry at desktop/phone sizes, both languages and unchanged checkpoints. Existing native help/layout checks also pass.
- Native and browser builds compile. `node tools/check-idle-review.cjs <test-server-url>` uses actual controls in isolated browser data, creates two idle owned lines, checks their Go-rendered map pixels, excludes an operating public line, exercises row focus and the Chinese 320px layout, then proves that accepted service removes exactly the affected highlight. It also checks idle fleets on active shared track, an idle through fleet over both sections and reload. [Results](idle-review-checks.json) pin the tested WASM build; screenshots are `idle-review-desktop.png` and `idle-review-phone-zh.png`.
- Review-only actions preserve both the checkpoint and exact durable campaign bytes. The player's native campaign was not opened or reset.

No full local suite or native OS interaction check was run. Physical devices and native OS input remain outside this verification.
