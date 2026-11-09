# Game state and storage

**Decision:** use a typed Go world, portable versioned JSON saves, and an embedded SQLite database for the private multiplayer server. Start with save schema **1**, host-record schema **1**, and database schema **1**. There are no released game saves to migrate.

This is the implementation contract for the [architecture](../ARCHITECTURE.md). The [reference Go types](contracts/state.go), [server port/types](contracts/server.go), [structural JSON Schema](contracts/save-v1.schema.json), [example save](contracts/fixtures/minimal-save-v1.json), and [initial SQL schema](contracts/server-v1.sql) make it concrete. The examples are compile/checkable design artifacts; they are not a game engine, a complete restore validator, or a running storage adapter.

## 1. Ownership and representation

Use one mutable `World` aggregate per game, owned by its application/coordinator. Domain state uses typed IDs, concrete records, and maps keyed by IDs. Callers receive isolated DTOs, not pointers to its maps or entities. The reference `World` illustrates ownership; production domain records gain private fields and invariant-preserving constructors/methods. JSON tags and schema compatibility belong to application-owned save DTOs.

| State | Representation and ownership |
| --- | --- |
| Scenario and balance | Immutable content-addressed bundles: cities, node/route geometry, engineering profiles, train/track models, costs, contract/offer terms, and scenario constraints |
| Business world | Companies; station/junction graph; ownership and access; projects; fleet/orders; services and acknowledged plans; finance; auctions/bids; reservations; cash journal; reports |
| Derived indexes | Adjacency, spatial/search indexes, OD candidates, cache entries, free-cash totals, and lookup tables; rebuild after validated restore |
| Server operations | Schedule/phase, locked resolution identity, canonical command receipts, audit events, authenticated sessions and invitations; excluded from portable saves and gameplay hashes |
| Local presentation | Camera, selection, audio, hints, replay cursor, and unsubmitted drafts; separate versioned UI records |

The immutable bundle manifest transitively hashes all referenced assets. Saves pin exact bundle IDs and SHA-256 values, plus simulation/demand/allocation/RNG algorithm IDs. They do not copy the basemap or corridor polylines. New construction selects qualified catalog geometry; a later free-waypoint feature would need an explicit geometry representation and compatibility decision.

### Quantities and identity

- `Money` is checked `int64` **game USD cents**. This is the game's common accounting unit, not a promise that supplied GDP estimates or real project costs share that price basis. No floating-point cash arithmetic.
- `FareRate` is checked `int64` in hundredths of a game cent (0.0001 USD) per passenger-km. Cent-sized fare steps would be too coarse for typical per-kilometre fares. Fare revenue rounds explicitly to `Money`.
- Distances are integer metres, durations integer seconds/minutes at named boundaries, passenger totals integer counts, and rates integer basis points. Monthly calculations use the actual Gregorian month.
- `GameMonth{Year, Month}` is the month still to resolve; it is not a local timestamp. Real UTC deadlines exist only in the server host record.
- World/company/city/node/section/project/service/trainset IDs have distinct Go types. Wire IDs are case-sensitive ASCII strings matching `[a-z][a-z0-9_-]{0,63}` and are unique within their entity collection. References are validated for their expected type and world/company ownership.
- The composition root supplies a world ID in the creation command. Scenario IDs are stable authored IDs. Dynamic domain IDs use a persisted monotonic `next_id` serial and type prefix; stable processing order makes allocation deterministic. Never reuse IDs after deletion, or derive identity from slice position/population rank. Overflow rejects the operation.

### Entity contracts

The complete field shapes are in [state.go](contracts/state.go); this table states their meaning.

| Record | Important semantics |
| --- | --- |
| `Company` | Ownership identity, status, home market, opening cash and current cash; optional restructuring deadline. The world pins `restructuring_window_months` at creation. |
| `Node`, `Station`, `StationAccess` | Explicit station/junction topology, immutable geometry reference, station capacity/book value/owner, and company-specific access terms; visual crossings do not create junctions |
| `TrackSection`, `TrackAccessOffer` | Referenced alignment and engineering, operational status, capacity, ownership, base access policy, and optional posted fee and offer revision |
| `TrackAccessAgreement` | Section, owner and operator IDs, accepted fee per train-kilometre, effective month, and optional first ineligible month. Separate station access still applies. |
| `Project` | New line or alignment, capacity, or station upgrade; geometry and targets, cost, start and ready months, status, and ordered installments. Pending multiplayer requests retain the reviewed competition fingerprint and protected funds. |
| `Trainset`, `TrainOrder`, `Lease` | Ownership, model/location/readiness, book value, pending deliveries and payment obligations. Status is delivery, operational, relocation, or retired (sold or withdrawn), and leased sets cannot be sold. A trainset cannot be assigned to disconnected services simultaneously |
| `ServiceSpec` | Ordered section traversals and called stops, assigned trainsets, desired daily return pairs, fare rate per passenger-km, and suspension; passing nodes do not automatically become stops |
| `CompanyPlan` | Private, complete desired service set for a named effective month, its own revision, and acknowledged standing safeguards |
| `Loan`, `Grant`, `Concession` | Pinned terms plus outstanding/paid/remaining obligations, project links, concession effective month (lead time) and expiry information needed to continue exactly |
| `Auction`, `Bid`, `Reservation` | Auction state, protected maximum bids and other earmarked cash; closed auctions retain outcome summaries while active bids/reservations are settled or removed together |
| `CashEntry` | Ordered cash posting with category, business month and reference; paired intercompany transfers share an identity and equal/opposite amounts |
| `MonthReport`, `OperatingDay` | Compact resolved results: company finance and requested/carried/alternate-served/rejected journeys, service boardings and revenue, and section capacity requested/granted with contributing services. The latest replay keeps leg timings, per-leg loads by direction and aggregate station activity; train positions at every animation frame are not saved. |

Ownership is `public`, `company`, or `estate`, with a company ID for the latter two. Public/acquired legacy sections retain regulated open access. A bankrupt owner's regulated lines use the public caretaker rule without deleting the estate's ownership/liability record.

The base policy `owner_only` can be supplemented by explicit agreements. An offer revision identifies a specific publication and advances when the owner republishes terms. Accept a multi-section offer atomically and pin the accepted fees independently of later offers. Validate that every section and owner matches, the requester is eligible, and rights for the same section and operator have disjoint effective periods. A section with an accepted right participates in shared allocation while the right applies.

Pending construction requests and their reservations are authoritative private state. Only committed or building projects enter the public construction frame. Derive the unordered pair from catalog endpoints. Check physical conflicts using catalog section identities and authored conflict groups. The competition fingerprint pins the published line and project identities reviewed by the player. Acceptance publishes the project and keeps its reservation. Rejection releases the reservation and returns the competing project to the requester.

`world.services` is the committed standing configuration carried from the last resolution. `world.plans` holds the latest acknowledged desired configurations for the current planning month; it does not overwrite published delivered service prematurely. Delivered frequency, ridership, and time belong to the resolved month's report. At resolution, freeze acknowledged plans, allocate service, publish results, and carry the accepted configuration forward. Unsubmitted client drafts never enter either collection.

Each service report retains its resolved `ServiceSpec`, so removing a service from a later plan does not erase its historical route/fare. Retain referenced retired trainsets and closed infrastructure records for the campaign; a historical report does not require its path to remain operational today. Loan principal arrears are a subset of outstanding principal, while interest arrears are an additional liability; do not count overdue principal twice. Lease arrears and grant obligations follow their pinned contract terms.

`state_revision` advances once for each accepted authoritative game mutation, including acknowledged plan changes. A company plan revision advances only for that company's plan changes. API stale-plan checks use month and that company's revision, not another company's private changes. `network_revision` invalidates geometry/connectivity caches when infrastructure changes. SQLite's `store_revision` is a separate compare-and-swap generation, including host-only writes and durable error receipts; it is not gameplay state or a client precondition.

## 2. Portable save v1

The envelope has exactly these fields:

```json
{
  "format": "wwhsrt-save",
  "schema_version": 1,
  "created_at_utc": "2026-11-08T00:00:00Z",
  "payload_sha256": "<64 lowercase hexadecimal characters>",
  "payload": {
    "state_revision": "0",
    "rules": "<RulesRef object>",
    "world": "<WorldV1 object>"
  }
}
```

The angle-bracket values above are explanatory placeholders; [minimal-save-v1.json](contracts/fixtures/minimal-save-v1.json) is a real structurally valid example with matching fixture-bundle hashes. Its tiny fictional bundles are contract-test data, not a qualified playable scenario.

Use UTF-8, schema-defined snake_case names, explicit nullable fields, and arrays rather than entity maps. All defined fields are required; an absent optional value is `null`, and an empty collection is `[]`, never `null`. Reject unknown fields for the selected version, duplicate JSON object keys, invalid UTF-8, trailing documents, unknown enum values, and integers outside their defined ranges.

Encode money, passenger/distance totals, seed, ID counters and revisions as **canonical decimal strings**. This preserves 64-bit precision through browser tooling. JSON Schema validates their lexical shape; typed Go decoding also checks their integer range. Small counts/month fields remain JSON integers. Replay leg seconds are nonnegative JSON-safe integers with scenario-bounded durations. No `map[string]any`/float-based generic decoding of authoritative numeric state. Go's JSON implementation requires an explicit duplicate-key check rather than relying only on `DisallowUnknownFields`. See the [Go JSON documentation](https://pkg.go.dev/encoding/json).

`payload_sha256` hashes the **exact encoded payload bytes**, including any whitespace inside that object. It detects accidental corruption, not malicious editing. The writer emits compact payload JSON. Reformatting only the outer envelope is harmless; reformatting the payload requires recomputing its checksum. Credentials and signing secrets are never embedded. Authoritative server state is loaded from the server database, not a client-uploaded save.

### Canonical checkpoints

Canonical state hashing is distinct from the file checksum: map the world to the canonical `PayloadV1`, encode with the pinned codec and fixed struct-field order, then SHA-256 the bytes. Exclude envelope creation time, host schedule, session/receipt data, UI state and caches. Include rule identities, accepted plans, counters, and authoritative financial state. Compare hashes only under the same codec and rule versions.

Sort entity sets by ID, plans by company, station access by station and company, and track agreements by section, operator, and effective month. Sort concessions by company, jurisdiction, and terms. Sort reports chronologically and their company, service and section rows by ID; sort replay stations by ID. Sort cash entries by sequence and set-like trainset and target IDs. Preserve meaningful path, stop, installment, replay-leg, departure, and RNG-word order. Reject duplicate IDs and noncanonical ordering during qualification.

Keep compact monthly reports/cash history for the configured campaign length (default 360 months), and replay detail for the latest resolved month only. The initial uncompressed save limit is **128 MiB**, with maximum nesting 64. Do not silently truncate records to fit. Qualify these limits against the proposed stress world; increasing a resource limit is a coordinated reader/writer change, not a data migration by itself.

### Restore pipeline

1. Enforce byte/depth limits and UTF-8; reject duplicate keys and trailing content. Read the format/version header without modifying live state.
2. Dispatch only to a supported schema reader; verify the raw payload checksum and complete structural schema.
3. Resolve exact scenario/balance hashes and supported algorithm IDs. Missing/incompatible data is an explicit error, not substitution with the latest data.
4. Validate semantics into an isolated candidate: months/IDs/references, company ownership, graph/path continuity, operational/readiness status, capacity/fleet eligibility, contract terms, project installment totals, and reservation/bid relationships.
5. Reconcile each company's cash to opening cash plus its journal; validate intercompany transfer pairs and debt/lease/grant/asset state against contract rules. Free cash derives from cash minus active reservations; do not persist a second free-cash balance. Validate solvency/arrears through the business rules rather than discarding protected obligations.
6. Validate plans, reports, replay references, monotonic counters/revisions, retention bounds and RNG mode. `rng: none` requires `rng_state: null`; future RNG algorithms must define exact word layout and validation.
7. Rebuild derived indexes, then atomically replace the in-memory world. Any earlier failure preserves the live campaign and original save bytes.

The reference [codec](contracts/codec.go) implements envelope/version, duplicate-key/UTF-8/size/depth, typed numeric decoding, unknown-field, and checksum checks. It deliberately does **not** implement the complete structural/semantic restore pipeline; the schema and domain validation gates must be wired before a production loader is considered complete.

## 3. Offline storage

Desktop uses a filesystem adapter and browser single-player uses IndexedDB. Both store the same portable envelope and keep UI records separately.

For filesystem slots, write a new generation to a temporary file in the slot directory, flush and close it, promote it to a new final filename, and make the directory update durable using the target platform's supported mechanism. Never overwrite the last good generation in place. Ignore temporary files on discovery. Use an adapter-local monotonic generation, not filesystem modification time, for ordering. Remove old autosave generations only after the new one is committed; retain the newest three plus independently named manual checkpoints. A corrupt final generation is reported with older recovery choices, not silently loaded into a live campaign. Native durability/promotion behavior must be qualified on each target: Go's [Rename documentation](https://pkg.go.dev/os#Rename) does not promise atomic replacement on every OS.

For IndexedDB, write the complete JSON string and perform slot retention in one read/write transaction. Request strict durability where supported and report completion only after transaction completion; qualify browser behavior rather than promising native filesystem guarantees. A quota/transaction failure preserves the prior slot. The [IndexedDB durability model](https://w3c.github.io/IndexedDB/#dom-idbtransactionoptions-durability) is the platform reference. Keep a single browser writer as required by the architecture, with export/import available when browser persistence is unavailable or evicted.

## 4. Multiplayer: SQLite

Use **SQLite through `database/sql` and the pure-Go `modernc.org/sqlite` driver**, isolated in the server storage adapter. Pin the driver and its SQLite version when wiring the server, after supported-OS/build/crash smoke tests; this design does not add an untested driver to the repository's module graph. The [driver documentation](https://pkg.go.dev/modernc.org/sqlite) describes its cgo-free implementation. Browser/native-client builds must not import this adapter.

One server process owns `server-data/wwhsrt.sqlite` on a persistent local disk. There is no separately administered database service. A process-level data-directory lock prevents accidentally running two coordinators against the same database; no active/active replicas or network-mounted database file in the first release. This is a good fit for the private 2–8-player scope and serialized coordinator. SQLite documents this application-server usage and its [single-writer transaction behavior](https://sqlite.org/lang_transaction.html).

Use one database connection (`SetMaxOpenConns(1)`, `SetMaxIdleConns(1)`) and a short serialized database-write path. Configure/read back `foreign_keys=ON`, `journal_mode=DELETE`, `synchronous=EXTRA`, and `busy_timeout=5000` on connection creation. SQLite's [EXTRA setting](https://sqlite.org/pragma.html#pragma_synchronous) adds a directory sync for DELETE-journal commits. Calculate forecasts/months outside database transactions; a database lock is not held while the simulation runs. WAL is unnecessary for this initial access pattern.

The [SQL bootstrap](contracts/server-v1.sql) contains five tables:

It uses SQLite [STRICT tables](https://sqlite.org/stricttables.html) (SQLite 3.37 or newer), application ID `0x57574852` and `user_version=1`. Bootstrap only when both identifiers are zero and the database is empty; otherwise validate the application ID and supported schema version before reading/writing. A file belonging to another application is not a schema-zero game database.

| Table | Contents |
| --- | --- |
| `worlds` | Current whole-world save JSON, host-record JSON, store generation, update timestamp |
| `operations` | Actor-scoped idempotency key, canonical typed game request and hash, exact authorized response and committed generation |
| `audit_events` | Ordered committed/rejected/phase events with correlation, business month, before/after hashes and scoped versioned payloads |
| `sessions` | World/company binding, host flag, hashed random bearer token, expiry/revocation; no plaintext token |
| `invitations` | Hashed random invite token, target company, expiry and remaining uses |

Keep entities inside the world snapshot instead of creating relational train/station/account repositories. This preserves the aggregate transaction boundary and keeps schema changes small. Operations are deduplication/reproduction records, not an event-sourced world; ordinary restore does not replay them. Audit events and operation history follow [DIAGNOSTICS.md](DIAGNOSTICS.md), are retained for the world's lifetime, and do not enter portable saves or gameplay hashes. This extends the still-unreleased SQL v1 bootstrap; there is no historical database migration to run. Sessions/invitations are operational identity data, excluded from game exports. Validate company bindings against the world because company rows live inside its snapshot.

V1 invitations target already-created company slots. Consume an invitation's remaining use and create the session in one identity-table transaction. Joining/reconnecting cannot create another company or starting-capital package. Creating an additional company is an explicit idempotent application command; an invitation is issued only after that company exists.

### Atomic command commit

The [application-owned store port](contracts/server.go) exposes load, operation lookup, and a compare-and-swap commit. Implement commit using one `BEGIN IMMEDIATE` transaction:

1. Authenticate/authorize first. Derive the actor namespace on the server. Look up `(world, actor, operation_key)` before stale-month checks: a matching request hash returns its recorded result; a changed payload with the same key returns `IdempotencyConflict`.
2. Apply a fresh operation to an isolated candidate world and validate it. Encode/check the candidate before beginning the database write.
3. Inside the transaction, recheck the operation identity and expected store generation. Update `worlds` with `WHERE store_revision = expected`; require exactly one changed row, incrementing without integer overflow.
4. Write the save, host state, operation receipt/canonical game request, and nonempty audit event group in that same transaction. Assign event indices in stable order within the new store generation. For an authenticated terminal business rejection, keep save/state revision unchanged but atomically store its error receipt, rejection event and new store generation. Unauthenticated failures do not create world receipts. Replayed operations return the existing outcome without appending another business event.
5. Commit successfully, then replace the coordinator's in-memory state and acknowledge/broadcast. Database errors before commit leave the old state usable. An ambiguous commit result requires reloading/checking the original receipt; stop further mutation until reconciled, rather than claiming the operation certainly failed.

Use a pinned `sql.Conn` for `BEGIN IMMEDIATE`, every statement, and `COMMIT`/`ROLLBACK`, returning it to the pool only afterward. Do not mix connection-scoped transaction statements with `DB.Exec` calls that could wait for the same sole connection. Validate that a receipt's committed generation equals the compare-and-swap result. Generate session tokens from 32 cryptographically random bytes and invites from at least 16; store only their SHA-256 hashes.

Retain operation receipts while the world exists, including old months; an offline player may retry long after a deadline. Bound/rate-limit requests instead of deleting deduplication history prematurely. Request hashes cover canonical typed command content and preconditions, excluding credentials and incidental transport timestamps. Player queries cannot choose the system actor or inspect another company's receipts.

### Monthly lock, commit and recovery

`ServerHostV1` stores a separate schema version, scheduler policy, phase (`planning`, `accepting`, `locked`, `paused`, `ended`), interval, UTC deadline, schedule revision, and optional locked resolution. The proposed default for a new world is 30 seconds. Resumed worlds use their stored interval. A locked resolution pins its business month, payload checksum/state revision and each company plan revision.

In `planning`, the opening world is available for read-only queries and the next deadline is null. Players can keep private drafts in client storage. A host start operation changes the phase to `accepting` and commits the first deadline one full interval after start, together with its receipt. Retrying start returns the same receipt and deadline. Convert the restructuring response minimum to whole game months at world creation. Store the larger of that count and the baseline window in `restructuring_window_months`.

For each online auction, persist the cutoff in host state when publishing the listing. Reject new or revised bids and withdrawals received at or after the cutoff using server time. Persist the seal and honor it after restart. An explicit host pause records the remaining time for open auctions. Resume restores that time. A locked resolution pins the sorted IDs of sealed auctions whose cutoff falls at or before that resolution's deadline. The domain settles those auctions from the pinned inputs, so replay uses the same bids and selection. The domain's `Auction.Closes` drives offline closure and represents the expected settlement month online. Persist the chosen auction IDs as part of the canonical lock request.

At the deadline, the world coordinator durably records `locked` before computing. Reject new commands for that closed planning window; callers refresh/retry with the next month after commit. Compute the month from the pinned snapshot in scratch state. Commit the next world, next deadline, accepting/ended phase, and the system resolution receipt atomically. Only then publish results. A crash before commit recomputes from the unchanged locked snapshot; a crash after commit returns the recorded resolution. Neither case advances twice.

Normal deadlines stay on the fixed schedule (`previous deadline + interval`). Restart after an outage completes a locked month or resolves at most one overdue month, then establishes a full interval from recovery. If a clock jump or exceptional delay would otherwise cause immediate repeated catch-up, apply the same bounded policy and log the adjusted schedule. Paused worlds do not catch up; explicit resume starts a full interval and records that action. These wall-clock policies do not enter canonical domain hashes.

Use SQLite's [backup API or VACUUM INTO](https://sqlite.org/backup.html) for a live consistent database backup, or close the server before copying its database. Back up the whole database and pinned data bundles; a lone world JSON would omit the receipts needed for recovery. A valid SQLite rollback journal must not be manually discarded. Corruption/checksum failures quarantine the affected world for explicit recovery rather than automatically rewinding an acknowledged game.

An admin portable game export contains the save envelope. Import creates a **new world ID**, schedule and admission credentials. For listed auctions carried into a new multiplayer world, disclose that they receive a fresh three-minute bidding window when the host starts. Preserve existing bids and reservations. Only validated whole-world saves can be imported.

## 5. Migrations

**No historical game-save migration is needed now.** The camera prototype's widget state, raw metro JSON, and rendering spike are not earlier game-save formats. Keep v1 marked unreleased until its first playable consumer is frozen; regenerate reference fixtures when this design changes before that point.

The SQL file is the initial `user_version=0 → 1` bootstrap for an empty database, not an invented legacy migration. Refuse an existing unknown/newer database version. Save schema, host-record schema and SQL schema are independently versioned; a region-camera change does not migrate campaign state.

After a released schema actually changes, add only the required adjacent migration (for example `save 1 → 2`) with real old/new fixtures. Migration is a pure conversion into a new candidate, followed by the full validation pipeline; preserve the original and write a separate new generation after success. Never initialize missing obligations, ownership or assets to convenient defaults merely to make a load succeed. A semantic/rule/data-version change needs an explicit compatibility decision, not just a JSON field rename.

Future SQL migrations run with the server stopped, a verified whole-database backup, and a single transaction updating `user_version` only on success. If save blobs also change, migrate all affected blobs/host records/receipt-response compatibility as part of that upgrade or keep the old reader; do not leave a mixed-version database. Test interruption, rollback and backup restoration. Do not implement generic migration registries, downgrade machinery, or legacy converters before a real supported transition exists.

## 6. Verification and remaining implementation

The focused reference Go checks compile the types, compare the generated structural schema/example, check exact wide-integer round trips, reject duplicate/trailing JSON and tampered/future-schema envelopes. The Python contract checker validates the schema/example and fixture hashes and exercises SQLite bootstrap, atomic world/receipt/audit rollback, generation conflicts, deduplication without duplicate events, close/reopen and backup consistency.

These checks do not implement world mapping, complete semantic restore validation, the selected Go SQL driver, production native/browser storage, authentication, process locking or a server coordinator. Those are implementation tasks with the contracts above fixed for the first slice. Qualify real driver/filesystem/browser fault recovery and abrupt-process/power-loss behavior before claiming production durability. Follow [TESTING.md](../TESTING.md) for the focused commands and remaining gates.
