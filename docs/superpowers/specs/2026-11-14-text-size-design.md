# Text size (native app)

Status: approved 2026-11-14 and implemented. Item 7 of Behaviour was added in planning and awaits
confirmation; the native hand check has not been done.
Size: a small project (about 100 font-size sites, the fixed chrome sizes, one
new setting, tests at each step).

## Intent

Text in the native app is hard to read on a large desktop monitor. The
reported case is a 32-inch 4K display at native scale (about 140 pixels per
inch), where the app's 9 to 16 px text is 2 to 3 mm tall. Every font size is a
fixed number in the code and there is no setting to change it.

Two things fix this: larger default sizes on desktop windows, and a player
setting that makes all text larger still.

## Goals

- No text below 14 px on desktop-width windows, and none below 12 px on
  phone-width windows.
- Body text on desktop-width windows is 16 px, and the steps above it keep
  their order.
- A saved text size setting with four steps that multiplies every size.
- Text is drawn at its real size, so it stays sharp at any display resolution.

## Non-goals

- The browser client. Browser zoom already scales its panels and its map. Its
  78 style rules below 14 px are a separate change.
- Scaling icons, map markers, line widths or hit targets beyond what the text
  inside them needs.
- Asking the display for its scale factor (the app draws at half resolution on
  Retina and scaled displays today). Separate issue.
- Choosing a default step from the monitor's size. The player picks.

## Anti-goals

- Text cut off or overlapping at any step, on any window size the app allows
  (320×480 and up).
- The ordinary plan needing to scroll at 100% on the two sizes where it fits
  today (390×844 and 1280×850).
- Any change to rules, saves, forecasts or the frame contract.
- Any change to what the browser client draws, including its map.

## Behaviour (decided)

1. **Two ladders, chosen by window width.** "Desktop" is a window at least
   800 px wide, the same width at which the layout leaves its compact form.

   | Today | 9–11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 20 |
   |---|---|---|---|---|---|---|---|---|---|
   | Desktop (800 px and wider) | 14 | 14 | 14 | 16 | 17 | 18 | 19 | 20 | 22 |
   | Phone (under 800 px) | 12 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 20 |

   Desktop: floor of 14, and everything from 14 up gains 2 px. Phone: floor of
   12, everything else unchanged. Panel text on phones is therefore unchanged;
   only the map's 9 and 11 px legends grow.
2. **Text size setting.** Company → Settings gains a text size control beside
   Appearance, with four steps: 100%, 125%, 150%, 200%. The step multiplies
   every size from the ladder in use. It is a device preference: saved across
   restarts and kept by New game.
3. **Chrome grows with the step.** The header, footer, command dock (top and
   left), tab strips, the default panel width and the minimum height of
   controls are multiplied by the step, so larger text is not clipped. At 100%
   they keep today's sizes.
4. **Fit.** At 100% the ordinary plan fits one screen wherever it fits today.
   Above 100% panels may scroll; nothing may be cut off.
5. **Map text.** Labels, tags, legends, the replay clock and the terrain scale
   follow the same ladder and step. Boxes drawn behind map text are sized from
   the text.
6. **Shrink-to-fit text.** The one value that shrinks to fit its box (workspace
   header) may go below the floor only when it would otherwise be cut off.

7. **Changed in planning (2026-11-14), awaiting confirmation.** Reading the
   chrome code forced three refinements:
   - **A step is the same as a smaller window.** At step k the layout is the
     100% layout of a window 1/k the size, with every rectangle and font
     multiplied by k. So breakpoints scale too: a 1500 px window at 200% uses
     the compact layout, as a 750 px window does at 100%.
   - **The ladder follows the layout,** not the raw window width: the desktop
     ladder when the layout is not compact, the phone ladder when it is. At
     100% this is the 800 px rule above. Above 100% the phone floor is at
     least 15 px, so the 14 px desktop goal still holds.
   - **A step needs room.** The chosen step is used only while the window
     divided by it is at least 320×480 (the smallest window the app allows).
     Otherwise the largest step that fits is used: a 1280×850 window gets at
     most 150%, a 390×844 window stays at 100%. The setting shows both the
     chosen and the applied step when they differ.
   - **Existing clipped buttons are fixed first.** Hand-placed buttons are
     44 px tall whatever their label: 77 cases today need more (for example
     "Choose endpoints on the map" needs 65 px at 1280×850). Larger text makes
     this worse, so the map tool row and dock cells grow to their tallest
     label.

## Approaches considered

| Approach | Trade-off | Outcome |
|---|---|---|
| Draw text at larger sizes; grow chrome to fit | Sharp at any resolution; touches every font site and the fixed chrome sizes | **Chosen** |
| Zoom the whole frame (draw smaller, stretch to the window) | A few lines of code, but at 150% a 4K window is drawn at 2560×1440 and stretched | Rejected: gives up the sharpness that makes small text readable |
| Multiply every pixel constant by a scale factor | Sharp, and scales markers and hit targets too; touches every constant in panels, layout and map | Rejected: cost far beyond the problem |

Ladder alternatives, measured with throwaway trials against the native layout
tests (panel text only):

| Ladder | Desktop-width windows | Phone-width windows |
|---|---|---|
| Floor 14 only | Fits | Fits |
| Floor 14, 14 and up +2 | Fits | Plan up to 24 px over; help strip overflows |
| Floor 14, only 14→16 | Fits, but 15 px text ends up below body text | Plan 21 px over |
| Everything ×14/12 | Plan 14 px over at 1280×850 | Plan 62 px over |

## Constraints

- One function owns the ladder and the step. Panels and the map renderer ask it
  for a size; no font size is written as a bare number at a call site.
- It belongs in `pkg/ui` (presentation logic, shared by `pkg/app` and
  `pkg/render`) and takes plain values, as `pkg/ui` does not import
  `pkg/mapview`.
- `pkg/render` is shared with the browser client. The browser host must keep
  today's sizes, so the new ladder is something the native host opts into, not
  the renderer's default.
- Layout rectangles come from `ui.Workspace`, which is also what tests and the
  region frame of the route list use. Chrome scaling goes through it so those
  stay consistent.
- Existing native tests keep passing without weakening their assertions.

## Left to the builder

- Names and signatures; how the step is stored in `ui.State` (zero value must
  mean 100%, so existing preference files load unchanged).
- How "desktop or phone" reaches the map renderer, which knows the map's
  rectangle, not the window's.
- The form of the setting control (a cycling button like Appearance, or
  smaller/larger buttons like Panel width) and its English and Simplified
  Chinese strings.
- Whether the panel width the player chose is stored before or after the step
  is applied.
- Tests: the ladder table above as a unit test; a guard that no panel or map
  text is drawn below the floor; the existing fit and overflow tests repeated
  at each step for 320×480, 390×844, 1280×850, 1920×1080 and 3840×2160; the
  preference surviving restart and New game; the browser host's sizes
  unchanged.

## Evidence

- Display: LEN T32p-20, 3840×2160, "looks like 3840×2160" (no scaling).
- Sizes in use today: panels 12 to 18 px (12 ×7, 13 ×22, 14 ×38, 15 ×12,
  16 ×6, 17 ×6, 18 ×5 through the label helper, plus about 20 direct sites);
  map 9 to 20 px.
- Fixed chrome in `ui.Workspace`: header 48, footer 64 (92 compact), top dock
  56 (52 or 96 compact), left dock 164, default panel width 440.
- The trials changed panel text only. Map text and chrome scaling above 100%
  are unmeasured.
