package contracts

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"
)

const MaxSaveBytes = 128 << 20

type SaveV1 struct {
	Format        string          `json:"format"`
	SchemaVersion uint16          `json:"schema_version"`
	CreatedAtUTC  string          `json:"created_at_utc"`
	PayloadSHA256 string          `json:"payload_sha256"`
	Payload       json.RawMessage `json:"payload"`
}

// EncodeV1 expects application-mapped DTOs with canonical collection ordering.
// It deliberately does not implement domain validation or world restoration.
func EncodeV1(payload PayloadV1, createdAtUTC string) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(body)
	encoded, err := json.Marshal(SaveV1{"wwhsrt-save", 1, createdAtUTC, hex.EncodeToString(hash[:]), body})
	if err != nil {
		return nil, err
	}
	if len(encoded) > MaxSaveBytes {
		return nil, errors.New("save too large")
	}
	return encoded, nil
}

// DecodeV1 checks version, duplicate keys, unknown fields, and payload integrity.
// Full required-field/shape and semantic validation are separate release gates.
func DecodeV1(data []byte) (PayloadV1, error) {
	var result PayloadV1
	if len(data) > MaxSaveBytes {
		return result, errors.New("save too large")
	}
	if !utf8.Valid(data) {
		return result, errors.New("save is not UTF-8")
	}
	if err := uniqueKeys(data); err != nil {
		return result, err
	}
	var header struct {
		Format        string `json:"format"`
		SchemaVersion uint16 `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return result, err
	}
	if header.Format != "wwhsrt-save" || header.SchemaVersion != 1 {
		return result, errors.New("unsupported save format/version")
	}
	var save SaveV1
	if err := decodeStrict(data, &save); err != nil {
		return result, err
	}
	if save.Format != "wwhsrt-save" || save.SchemaVersion != 1 {
		return result, errors.New("unsupported save format/version")
	}
	hash := sha256.Sum256(save.Payload)
	if save.PayloadSHA256 != hex.EncodeToString(hash[:]) {
		return result, errors.New("save payload checksum mismatch")
	}
	if err := decodeStrict(save.Payload, &result); err != nil {
		return result, err
	}
	return result, nil
}

func decodeStrict(data []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}

func uniqueKeys(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	var value func(int) error
	value = func(depth int) error {
		if depth > 64 {
			return errors.New("save nesting too deep")
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		delim, composite := t.(json.Delim)
		if !composite {
			return nil
		}
		switch delim {
		case '{':
			seen := map[string]bool{}
			for d.More() {
				key, err := d.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return errors.New("invalid object key")
				}
				if seen[name] {
					return fmt.Errorf("duplicate JSON field %q", name)
				}
				seen[name] = true
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		case '[':
			for d.More() {
				if err := value(depth + 1); err != nil {
					return err
				}
			}
		default:
			return errors.New("unexpected JSON delimiter")
		}
		_, err = d.Token()
		return err
	}
	if err := value(0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}
