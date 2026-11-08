package contracts

import (
	"context"
	"encoding/json"
)

// ServerHostV1 is operational state, excluded from portable saves and game hashes.
type ServerHostV1 struct {
	SchemaVersion    uint16            `json:"schema_version"`
	SchedulerPolicy  string            `json:"scheduler_policy"`
	Phase            string            `json:"phase"` // accepting, locked, paused, ended
	IntervalSeconds  uint32            `json:"interval_seconds"`
	NextDeadlineUTC  *string           `json:"next_deadline_utc"`
	ScheduleRevision uint64            `json:"schedule_revision,string"`
	Resolution       *LockedResolution `json:"resolution"`
}

type LockedResolution struct {
	ID                  string       `json:"id"`
	Month               GameMonth    `json:"month"`
	LockedStateRevision uint64       `json:"locked_state_revision,string"`
	LockedPayloadSHA256 string       `json:"locked_payload_sha256"`
	Plans               []LockedPlan `json:"plans"`
}

type LockedPlan struct {
	CompanyID CompanyID `json:"company_id"`
	Revision  uint64    `json:"revision,string"`
}

type StoredWorld struct {
	StoreRevision int64
	SaveJSON      []byte
	HostJSON      []byte
}

type OperationReceipt struct {
	Actor                  string // derived from authentication, or the scheduler's system actor
	Key                    string
	RequestSHA256          string
	RequestJSON            json.RawMessage // canonical typed game command; no credentials
	ResponseJSON           json.RawMessage
	CommittedStoreRevision int64
	CreatedAtUTC           string
}

type AuditEvent struct {
	Kind              string
	Scope             string // public, company, admin; projected views still authorize fields
	CompanyID         *CompanyID
	CorrelationID     string
	Month             GameMonth
	BeforeStateSHA256 string
	AfterStateSHA256  string
	PayloadVersion    uint16
	PayloadJSON       json.RawMessage
}

type StoreCommit struct {
	WorldID               WorldID
	ExpectedStoreRevision int64
	SaveJSON              []byte
	HostJSON              []byte
	Receipt               *OperationReceipt
	AuditEvents           []AuditEvent // nonempty; adapter assigns ordered event_index
	UpdatedAtUTC          string
}

// WorldStore belongs to the application layer. SQL details stay in its adapter.
// Commit atomically compares/increments revision and writes state/receipt/audit.
type WorldStore interface {
	Load(context.Context, WorldID) (StoredWorld, error)
	LookupOperation(context.Context, WorldID, string, string) (OperationReceipt, bool, error)
	Commit(context.Context, StoreCommit) (int64, error)
}
