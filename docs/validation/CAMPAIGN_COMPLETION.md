# Campaign completion feedback

[Local verification ledger](campaign-completion-checks.json): focused native
checks passed in 9.285 seconds; all eight isolated browser footer cases passed at
575 MiB peak combined checker RSS. Native and web bundles were rebuilt.

At 360 resolved months, native and browser clients replace the time controls with
a persistent **Campaign complete** notice, **30 years · Time stopped.**, and a
**New game** action. The action opens the existing destructive-reset confirmation;
opening or cancelling it does not reset the campaign. Reports and other tabs
remain available. Completion derives from the current frame, so it appears on
reopening a completed save, survives tab changes and muted onboarding, and clears
after starting a new game. English and Simplified Chinese are included.

Focused verification:

```sh
GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/ui -run '^TestChineseCatalogAndAliases$' -count=1
GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -run '^TestNative(CampaignCompletion|NewGameConfirmation|PanelsFitAvailableWidth|WrappedLabelsFitWithoutLosingText)' -count=1
NODE_OPTIONS=--max-old-space-size=1536 node tools/check-campaign-completion.cjs --layout
```

The native widget regression covers the final-month boundary, completed-frame
reconstruction, tab changes, muted help, confirmation/cancellation, and unchanged
exported campaign bytes. Notice text fits 320, 390, 800 and 1440 px layouts in both
languages. The browser check exercises the production completion renderer's
boundary, reload/reset and busy/composition states, then renders the actual footer
HTML, CSS and font at those eight language/width combinations. It checks text
clipping, viewport bounds and overlapping controls under the 2 GiB combined checker
RSS cap.

This isolated browser check does not start WASM or qualify the complete browser
application. The existing full-client startup memory limitation remains. Native
OS input and physical phones were not requalified; the running user campaign was
not changed. Build artifacts must be restarted/reloaded to show the updated UI.
