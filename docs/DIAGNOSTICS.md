# Diagnostic logging and event history

Pervasive, structured event logging is a product requirement for debugging after the fact. Record enough context to connect a player's action to validation, authoritative commit, month resolution, storage and visible feedback. This is the implementation contract; runtime logging/adapters are not implemented yet.

## Two complementary records

**Diagnostic logs** explain execution, failures and timings. Use Go `log/slog` JSON output through application/adapter decorators, with bounded local files on native/server hosts and a bounded IndexedDB diagnostic store in the browser. The domain returns diagnostic facts/results; it does not perform log I/O or read the wall clock. Logs and their verbosity do not enter campaign hashes or affect outcomes.

**Durable audit events and operation records** explain authoritative changes. On the server, store canonical typed game requests with their request hashes and outcomes in `operations`, and append ordered `audit_events` in the same SQLite transaction as the world, schedule and receipt. A committed change cannot lose its audit entry because a separate file logger failed. A retry returns the existing receipt and emits no duplicate business event. This augments snapshots; restoring the game does not require replaying an event-sourced database.

Diagnostic logging failures never change simulation decisions. A failed SQLite audit write is a failed authoritative transaction, so no success acknowledgment is sent; it follows the ordinary store-failure/retry protocol. A diagnostic sink must not become an external service dependency for offline play.

## Coverage

| Area | Required events/context |
| --- | --- |
| Startup/lifecycle | Build/protocol/data/algorithm identities, platform, storage opened, resume/suspend, audio availability, shutdown |
| Commands | Received/validated/rejected/committed/replayed, command kind, entity IDs, stable constraint codes and authoritative revisions |
| Construction/fleet | Quote breakdown/rule identity, commitments, installments, cancellation, delivery, completion and access/ownership changes |
| Monthly resolution | Durable lock identity; phase start/end, input/output hashes, durations, aggregate market/capacity/allocation/cash totals, invariant failures, completion |
| Multiplayer | Session lifecycle, redacted admission failures, reconnect, stale plans, uncertain acknowledgments, deadlines, clock changes and bounded recovery |
| Storage | Save/load/migration/backup start and outcome, schema/data versions, checksum, generation, transaction outcome and corruption/limit errors |
| UX/render/audio | Intent and acknowledgment correlation, meaningful feedback scheduling/cancellation, input/lifecycle failures and slow frames; never one event per frame/train position |

INFO covers meaningful lifecycle changes, committed actions and month summaries. Expected business-rule rejections use structured outcome codes rather than alarming stack traces. WARN covers recoverable problems; ERROR covers failed storage, invariant violations and crashes. DEBUG adds bounded forecast/allocation reasoning, cache behavior and selected service/OD traces. Use bounded asynchronous diagnostic queues so file/browser sink latency does not block the coordinator; drop verbose entries first under pressure and expose counts. Rate-limit noisy repeated failures and log the suppressed count. Do not sample away authoritative audit events.

Each applicable event carries a schema/version, event name, severity, UTC timestamp, process/run ID, world/company ID, business month, request/correlation ID, operation key, relevant entity IDs, store/state/plan/network revisions, and stable outcome/error code. Stage timings use a host monotonic clock. Audit order is `(world_id, store_revision, event_index)`, independent of wall-clock ordering. State-changing events include before/after canonical payload hashes. Global startup events may omit world/company/month fields.

A generic line such as “something failed” is insufficient. For example, a denied departure increase should identify the service, requested/granted return pairs, constraining section/station and rule version. Passenger allocation traces explain aggregate choices without logging individual passengers.

## Retention, privacy and reproduction

- Keep server audit/operation records for the world's lifetime. Store complete canonical **game command** DTOs, not raw HTTP headers or arbitrary request bodies. Initial limits are 1 MiB per command and 32 KiB for a mutation acknowledgment; large query results are not receipts. Audit payloads are capped at 8 KiB per event and summarize large sets with explicit omitted counts.
- Initial rolling diagnostic caps: 16 MiB on desktop (four 4 MiB files), 128 MiB on the server (eight 16 MiB files), and 8 MiB in the browser, with a 30-day age cap. These are bounded diagnostic buffers, separate from durable audit retention. Drop/coalesce verbose diagnostics under pressure and expose dropped counts. Never discard authoritative retry history to meet a log-file cap.
- Do not log bearer tokens, invite codes, credentials, raw authorization headers or whole save blobs. Diagnostic fields are allowlisted and bounded. Financial plans, bids and detailed forecasts remain company-private. The host's complete audit store is admin-only; player activity views receive separately authorized/redacted projections, not raw event JSON.
- No gameplay telemetry is uploaded by default. **Export diagnostics** produces a local support bundle for deliberate sharing; it reports included company/world scope and any redaction. Never silently send a save, rival plan or diagnostic bundle elsewhere.
- A reproducible bundle contains build/data/algorithm identities, a suitable checkpoint and ordered canonical commands after it, relevant audit/diagnostic records, and the first divergent hash. Server snapshots plus retained command receipts provide that history. In offline play, keep a bounded command journal alongside autosave generations; if storage failure/rotation leaves a gap, mark reproduction incomplete rather than claiming a replayable history.
- Reproduce through the public application API with the pinned inputs. Host scheduling observations accompany the command log for investigating deadlines; wall-clock data does not become a domain input during canonical replay.

Normal users see clear actionable errors and an export control under settings, not a permanent debug console. Developers/admins can enable detailed traces. Trace collection, rendering effects and audio preferences must leave canonical results unchanged.

## Verification

Test correlation from intent through durable outcome; exactly one audit event group after retry; rollback of world/receipt/audit together; sensitive-field exclusion and authorized exports; rotation/drop counters; and equal checkpoints with logging disabled, normal, verbose or a failed diagnostic sink. An abrupt-process recovery test must recover the committed audit trail as well as the world. The current SQL reference checker covers transaction-level audit consistency; it does not establish production logging or crash recovery.
