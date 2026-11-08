"""Focused JSON/SQLite contract checks; not a game or production-driver test."""

import copy
import datetime
import hashlib
import importlib.metadata
import json
from pathlib import Path
import sqlite3
import tempfile

import jsonschema


ROOT = Path(__file__).resolve().parents[1]
CONTRACTS = ROOT / "docs/contracts"
FIXTURES = CONTRACTS / "fixtures"


def digest(data):
    return hashlib.sha256(data).hexdigest()


def encode(payload):
    body = json.dumps(payload, ensure_ascii=False, separators=(",", ":"))
    return json.dumps({
        "format": "wwhsrt-save", "schema_version": 1,
        "created_at_utc": "2026-11-08T00:00:00Z",
        "payload_sha256": digest(body.encode()), "payload": payload,
    }, ensure_ascii=False, separators=(",", ":"))


def configure(db):
    for sql in ("PRAGMA foreign_keys=ON", "PRAGMA journal_mode=DELETE",
                "PRAGMA synchronous=EXTRA", "PRAGMA busy_timeout=5000"):
        db.execute(sql)
    assert db.execute("PRAGMA foreign_keys").fetchone()[0] == 1
    assert db.execute("PRAGMA journal_mode").fetchone()[0] == "delete"
    assert db.execute("PRAGMA synchronous").fetchone()[0] == 3


def main():
    schema_bytes = (CONTRACTS / "save-v1.schema.json").read_bytes()
    schema = json.loads(schema_bytes)
    jsonschema.Draft202012Validator.check_schema(schema)
    validator = jsonschema.Draft202012Validator(schema, format_checker=jsonschema.FormatChecker())
    save_text = (FIXTURES / "minimal-save-v1.json").read_text()
    save = json.loads(save_text)
    payload = json.loads((FIXTURES / "minimal-payload.json").read_bytes())
    validator.validate(save)
    assert save["payload"] == payload
    start = save_text.index('"payload":') + len('"payload":')
    while save_text[start].isspace():
        start += 1
    _, end = json.JSONDecoder().raw_decode(save_text, start)
    assert save["payload_sha256"] == digest(save_text[start:end].encode())
    for key in ("scenario", "balance"):
        assert payload["rules"][key]["sha256"] == digest((FIXTURES / f"{key}.json").read_bytes())
    bad = []
    value = copy.deepcopy(save); del value["payload"]["world"]["nodes"]; bad.append(value)
    value = copy.deepcopy(save); value["payload"]["world"]["month"]["month"] = 13; bad.append(value)
    value = copy.deepcopy(save); value["payload"]["world"]["companies"][0]["cash"] = 100; bad.append(value)
    value = copy.deepcopy(save); value["payload"]["world"]["companies"][0]["cash"] = "-0"; bad.append(value)
    value = copy.deepcopy(save); value["payload"]["world"]["stations"] = None; bad.append(value)
    value = copy.deepcopy(save); value["payload"]["world"]["unknown"] = True; bad.append(value)
    value = copy.deepcopy(save); value["schema_version"] = 2; bad.append(value)
    for value in bad:
        assert not validator.is_valid(value), "invalid save accepted by structural schema"

    ddl = (CONTRACTS / "server-v1.sql").read_text()
    world_id = payload["world"]["id"]
    host = {"schema_version": 1, "scheduler_policy": "fixed-interval-bounded-recovery-v1",
            "phase": "accepting", "interval_seconds": 900,
            "next_deadline_utc": "2026-11-08T00:15:00Z", "schedule_revision": "0", "resolution": None}
    now = "2026-11-08T00:00:00Z"
    checks = ["generated JSON schema/example and fixture bundle hashes",
              "missing/unknown fields, null arrays, month bounds, wide-number syntax, future version"]

    with tempfile.TemporaryDirectory(prefix="wwhsrt-state-contract-") as tmp:
        path = Path(tmp) / "server.sqlite"
        db = sqlite3.connect(path, isolation_level=None)
        assert db.execute("PRAGMA user_version").fetchone()[0] == 0
        db.executescript(ddl)
        configure(db)
        assert db.execute("PRAGMA user_version").fetchone()[0] == 1
        assert db.execute("PRAGMA application_id").fetchone()[0] == 0x57574852
        db.execute("INSERT INTO worlds VALUES (?,0,?,?,?)", (world_id, encode(payload), json.dumps(host), now))

        def snapshot():
            return (db.execute("SELECT * FROM worlds").fetchall(),
                    db.execute("SELECT * FROM operations ORDER BY actor,operation_key").fetchall())

        def commit(expected, candidate, host_state, actor, key, request_hash, response, fail_at=None):
            validator.validate(json.loads(encode(candidate)))
            db.execute("BEGIN IMMEDIATE")
            try:
                prior = db.execute("SELECT request_sha256,response_json,committed_store_revision FROM operations "
                                   "WHERE world_id=? AND actor=? AND operation_key=?", (world_id, actor, key)).fetchone()
                if prior:
                    if prior[0] != request_hash:
                        raise ValueError("IdempotencyConflict")
                    db.execute("COMMIT")
                    return prior[2], json.loads(prior[1])
                changed = db.execute("UPDATE worlds SET store_revision=store_revision+1,save_json=?,host_json=?,updated_at_utc=? "
                                     "WHERE world_id=? AND store_revision=?",
                                     (encode(candidate), json.dumps(host_state), now, world_id, expected))
                if changed.rowcount != 1:
                    raise ValueError("StoreRevisionConflict")
                if fail_at == "state":
                    raise RuntimeError("injected failure after state update")
                db.execute("INSERT INTO operations VALUES (?,?,?,?,?,?,?)",
                           (world_id, actor, key, request_hash, json.dumps(response), expected + 1, now))
                if fail_at == "receipt":
                    raise RuntimeError("injected failure after receipt insert")
                db.execute("COMMIT")
                return expected + 1, response
            except BaseException:
                db.execute("ROLLBACK")
                raise

        changed_payload = copy.deepcopy(payload)
        changed_payload["state_revision"] = "1"
        changed_payload["world"]["plans"][0]["revision"] = "1"
        actor = "company:co_player"
        request = digest(b"apply-plan-month-2025-01-revision-0")
        for point in ("state", "receipt"):
            before = snapshot()
            try:
                commit(0, changed_payload, host, actor, "op_1", request, {"accepted": True}, point)
            except RuntimeError:
                pass
            else:
                raise AssertionError("injected transaction did not fail")
            assert snapshot() == before
        assert commit(0, changed_payload, host, actor, "op_1", request, {"accepted": True})[0] == 1
        before = snapshot()
        assert commit(0, payload, host, actor, "op_1", request, {"accepted": False}) == (1, {"accepted": True})
        assert snapshot() == before
        for key, request_hash in (("op_1", digest(b"changed request")), ("op_2", request)):
            try:
                commit(0, payload, host, actor, key, request_hash, {})
            except ValueError:
                pass
            else:
                raise AssertionError("conflict accepted")
            assert snapshot() == before
        checks.append("SQLite v1 bootstrap, configured durability, atomic rollback at state/receipt boundaries, CAS and actor/key deduplication")

        db.close()
        db = sqlite3.connect(path, isolation_level=None); configure(db)
        assert snapshot() == before
        assert commit(1, changed_payload, host, actor, "rejected", digest(b"invalid service"), {"error": "InsufficientFleet"})[0] == 2
        assert json.loads(db.execute("SELECT save_json FROM worlds").fetchone()[0])["payload"] == changed_payload
        checks.append("close/reopen preserves receipts; durable rejection does not change game payload")

        locked = copy.deepcopy(host)
        locked["phase"] = "locked"
        locked["resolution"] = {"id": "resolve_2025_01", "month": {"year": 2025, "month": 1},
                                "locked_state_revision": "1", "locked_payload_sha256": json.loads(encode(changed_payload))["payload_sha256"],
                                "plans": [{"company_id": "co_player", "revision": "1"}]}
        assert commit(2, changed_payload, locked, "system", "lock:2025-01", digest(b"lock 2025-01"), {"locked": True})[0] == 3
        next_payload = copy.deepcopy(changed_payload)
        next_payload["state_revision"] = "2"
        next_payload["world"]["month"] = {"year": 2025, "month": 2}
        next_payload["world"]["plans"][0]["effective_month"] = {"year": 2025, "month": 2}
        next_host = copy.deepcopy(host); next_host["next_deadline_utc"] = "2026-11-08T00:30:00Z"
        before = snapshot()
        try:
            commit(3, next_payload, next_host, "system", "resolve:2025-01", digest(b"resolve 2025-01"), {"month": "2025-02"}, "receipt")
        except RuntimeError:
            pass
        assert snapshot() == before
        assert commit(3, next_payload, next_host, "system", "resolve:2025-01", digest(b"resolve 2025-01"), {"month": "2025-02"})[0] == 4
        before = snapshot()
        assert commit(3, payload, locked, "system", "resolve:2025-01", digest(b"resolve 2025-01"), {}) == (4, {"month": "2025-02"})
        assert snapshot() == before
        checks.append("locked-month failure retains lock/checkpoint; resolved world, deadline and completion receipt commit together and retry once")

        db.execute("INSERT INTO sessions VALUES (?,?,?,?,?,0)", ("session_a", world_id, "co_player", bytes(range(32)), "2026-12-01T00:00:00Z"))
        backup = sqlite3.connect(Path(tmp) / "backup.sqlite", isolation_level=None)
        db.backup(backup)
        assert backup.execute("SELECT * FROM worlds").fetchall() == before[0]
        assert backup.execute("SELECT * FROM operations ORDER BY actor,operation_key").fetchall() == before[1]
        assert backup.execute("SELECT count(*) FROM sessions").fetchone()[0] == 1
        assert backup.execute("PRAGMA integrity_check").fetchone()[0] == "ok"
        assert backup.execute("PRAGMA foreign_key_check").fetchall() == []
        backup.close(); db.close()
        checks.append("SQLite backup preserves consistent world/receipts/identity rows; integrity and foreign-key checks")

    result = {"date": datetime.datetime.now(datetime.timezone.utc).date().isoformat(),
              "scope": "Reference JSON schema and SQLite transaction contract via Python sqlite3; not the production Go adapter or domain restore validator",
              "sqliteVersion": sqlite3.sqlite_version, "jsonschemaVersion": importlib.metadata.version("jsonschema"),
              "schemaSHA256": digest(schema_bytes), "sqlSHA256": digest(ddl.encode()), "checks": checks, "passed": True}
    (ROOT / "docs/validation/state-contract-checks.json").write_text(json.dumps(result, indent=2) + "\n")
    print(json.dumps(result, indent=2))


if __name__ == "__main__":
    main()
