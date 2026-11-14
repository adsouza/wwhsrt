# Native representative-day playback

Verified on 2026-11-12.

Clicking Play rebuilt the native panel with `slider.Current` assigned directly.
Ebiten UI v0.7.3 compares that value with its default change baseline during
`Slider.Render`, emits a Changed event and invokes the seek handler. That handler
immediately paused the replay. This was a playback interruption, rather than an
imperceptibly fast operating day.

The slider now uses `InitialCurrent`, which initializes both values. Its seek
handler ignores unchanged values and queued events from a replaced slider.
Playback timing and business state are unchanged: at 60 updates per second the
clock advances ten game minutes per real second, repeating after 144 seconds.

Focused verification:

- `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/app -run '^TestNative(Replay|NetworkViewsFit)' -count=1`
  passed. The new regression failed before the fix with “initial slider render
  paused playback.” It exercises actual slider rendering/deferred events after
  Play and panel rebuilds, stale events, pause retaining elapsed time, scrub and
  resume, English/Simplified Chinese and byte-identical campaign exports.
  Existing native network-layout coverage passed alongside it.
- `GOMEMLIMIT=768MiB GOMAXPROCS=2 go test -p 1 ./pkg/render -run '^TestRepresentativeTrainTimingDwellReverseAndPick$' -count=1`
  passed for train position, station dwell, reverse direction, picking and repeat.
- Native app rebuilt with `sh tools/build-native.sh`.

This pass uses offscreen widget rendering and synthetic widget events, not OS
mouse interaction. No browser source changed, so browser workflow checks were
not repeated. No full root suite or TypeScript compilation ran. The rebuilt
native app requires a restart to load the fix; the player's save was not changed.
