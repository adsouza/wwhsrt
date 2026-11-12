# Visual identity, motion and sound

The UI should feel cheerful, bright and inviting. This is a product requirement. The [theme manifest](../data/ui/theme.v1.json) defines the initial palette: vivid railway blue, tropical teal, sunflower yellow, coral and violet, supported by pale sky/warm-white surfaces, fresh green land and readable deep-ink text. Dark appearance keeps those lively accents on navy surfaces. Keep tables and financial text calm and clear while the railway provides color and motion.

The first-business implementation now provides **System / Light / Dark** under Company → Appearance. Its shared runtime tokens live in [pkg/ui/theme.go](../pkg/ui/theme.go); the earlier manifest remains the prototype/design palette. Native controls and the browser host use the runtime palette for panels, inputs, warnings, dialogs, active states and map rendering. Relief imagery is dimmed in dark mode while contours, water, terrain labels, station markers and rail colours remain distinct. System follows live OS/browser preference (light if unknown), and explicit overrides persist locally across restart and New game. Appearance is excluded from campaign snapshots and checkpoints. [Theme verification](validation/theme-checks.json) records actual browser rendering and interaction; macOS system-dark rendering was also visually inspected.

Bright colors have stable roles: blue for primary actions/owned rail, teal for public/open access, amber for construction, violet for rivals, coral for drafts. Pair them with labels, icons and different line styles. Bright fill colors are not automatically suitable for small text; use the dedicated dark/light text tokens. Required text contrast targets are 4.5:1 for normal text, 3:1 for large text, and 3:1 for necessary graphical/control boundaries. Use a contrasting rail casing when the accent alone cannot meet its background. Qualify actual rendered states, including focus, selection and disabled controls.

## Motion grounded in the game

Presentation consumes authoritative frames and resolved summaries. A preview, unacknowledged command or animation cannot make infrastructure operational. No animation/audio completion callback changes money, capacity, construction time or the business month.

| Moment | Visual feedback | Sound direction |
| --- | --- | --- |
| Project committed | Planned corridor becomes an amber dashed work line; endpoint work markers appear after acknowledgment | Brief, light confirmation/metallic placement cue |
| Construction advances | Brief sleeper/rail reveal or short work-front sweep on the selected visible project, progressing only after a resolved month | Soft rail clinks/tamping and restrained machinery texture, limited to the focused project |
| Bridge/tunnel work | Relevant project symbol/segment changes with the project stage; no unexplained leap across the barrier | Subtle material/work texture, without a constant loud drilling loop |
| Line opens | Construction styling changes to its operational style; short station/route highlight and opening receipt | Warm, short completion chime |
| Train departs/runs | Small train markers follow the resolved path, accelerate/decelerate according to supplied replay timing and stop at called stations | Modern electric-motor rise, wheel/rail texture and a soft pass-by whoosh; avoid steam-engine conventions for HSR |
| Service/arrival selected | Route and called stops highlight; a small arrival/dwell acknowledgment where timing is available | Quiet arrival cue or station ambience, never necessary to understand the result |

V1 construction timing currently comes from accepted project start/ready months and status, not a metre-by-metre construction simulation. Label that view **scheduled construction progress**. A reveal is an illustration of the timeline; it must not claim that a particular partial section can carry trains. Use authored completed-work facts when the model later provides them. Stop the work reveal before its opening marker until the authoritative completion transition occurs.

The representative operating day remains explicitly labeled as replay. Its train positions use resolved paths, departures and leg timings plus qualified dwell information; view speed changes only illustration. Render a bounded set of train markers, simplify at world scale, and focus detail on selected/nearby services. Do not animate every passenger or require the player to watch a replay to advance.

Use short purposeful transitions: roughly 120–200 ms for control feedback, 250–400 ms for selection/focus changes, and at most about 800 ms for a construction/opening flourish. These are initial tuning targets. Never block an action until the flourish finishes, flash a whole screen, or keep every marker pulsing. Reconnecting after many months shows the latest state and one catch-up summary rather than playing a queue of old construction effects/chimes.

## Sound and accessibility

Provide master mute and independent music, effects and ambience levels, plus construction and operating-ambience toggles. Initial effects are gentle; warning sounds remain brief. Essential warnings always have text/visual equivalents. Remember these as local UI preferences, not campaign rules.

Only foreground, visible/focused railway activity contributes contextual sound. Cap concurrent voices, combine dense activity into ambience, and avoid a global chorus of every train or construction site. Distance/zoom/selection can attenuate sounds. Pause/scrub or stop replay ambience with the replay; fast-forward does not emit a sound for every skipped departure. Opening sound triggers once per visible acknowledged event; reloading or retrying a command does not play it again automatically.

Persist the latest seen business-event/cue identities in local UI state so an opening receipt does not chime repeatedly after reload. Running-train cues instead belong to an explicit playback session: starting replay again may produce its contextual sounds, while seeking/resume does not burst-play skipped cues. These presentation cursors never enter the portable world snapshot.

Start/unlock browser audio through a user interaction and handle unavailable audio quietly. Suspend/mute activity on hidden/background views and resume without a backlog. The server never initializes audio. Sound assets and their licensing/provenance are tracked and bundled for offline play; actual recordings/synthesis choices and mixing still need an asset pass.

Honor system reduced-motion preferences by default and expose an explicit override. Reduced motion replaces route-reveal sweeps and moving trains with stable progress/state changes, static positions and aggregate counts. Independent sound controls still apply. No repeated flashes or forced camera travel. On smaller/slower devices, reduce decorative particles/marker density before removing management information.

## Implementation and verification

`pkg/render` illustrates frames; `pkg/audio` receives presentation cue intents through a bounded mixer. Correlate cues to stable resolved event IDs to deduplicate them. Diagnostics may record scheduling/cancellation/failure summaries but not an event for every animation frame or audio sample. Neither presentation component can mutate the domain or consume campaign RNG.

Validate light/dark contrast, color-vision alternatives and real phone readability. Compare canonical checkpoints with effects/sound on/off, reduced motion and different replay speeds. Test cue deduplication after retries/reload/reconnect, no backlog on resume, mute/volume controls, browser audio unlock/failure, background suspension, bounded marker/voice counts and CPU/memory. Physical phones, audio assets and actual game effects remain to implement and qualify; a recolored prototype is visual direction, not proof those effects exist.
