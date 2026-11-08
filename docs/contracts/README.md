# State contract references

These files make the [state/storage design](../STATE_AND_STORAGE.md) reviewable and checkable before the game module is implemented. They are not a production domain, storage adapter, or complete restore validator.

- [state.go](state.go): domain ownership shape and complete portable save DTO field definitions.
- [server.go](server.go): server-only host state and application-owned transactional store port.
- [codec.go](codec.go): partial reference codec for envelope/version, typed numbers, duplicate/unknown fields, bounds and checksum handling. Production restore additionally requires the full structural and semantic validation pipeline.
- [schema.go](schema.go): generates [save-v1.schema.json](save-v1.schema.json) from the reference DTOs; rejects missing/unknown fields and describes lexical/range/enum structure.
- [fixtures](fixtures): tiny fictional data bundles and a complete structural payload/save example, with actual matching hashes. They are not a qualified playable scenario.
- [server-v1.sql](server-v1.sql): SQLite database-v1 bootstrap, not a legacy save migration.

Follow [TESTING.md](../../TESTING.md#state-and-save-contracts) for focused checks and explicit fixture/schema regeneration. After the first released schema is frozen, compatibility changes require real migration fixtures rather than overwriting old examples.
