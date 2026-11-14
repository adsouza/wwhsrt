package contracts

import (
	"encoding/json"
	"reflect"
	"strings"
)

// JSONSchemaV1 generates the reviewable structural schema from the reference
// wire structs. Semantic graph, rights, accounting and bundle validation remain
// application/domain responsibilities, independently of this schema.
func JSONSchemaV1() ([]byte, error) {
	defs := map[string]any{}
	var schema func(reflect.Type, bool) map[string]any
	schema = func(t reflect.Type, quoted bool) map[string]any {
		if t == reflect.TypeFor[json.RawMessage]() {
			return schema(reflect.TypeFor[PayloadV1](), false)
		}
		if t.Kind() == reflect.Pointer {
			return map[string]any{"anyOf": []any{schema(t.Elem(), quoted), map[string]any{"type": "null"}}}
		}
		if quoted {
			pattern := "^(0|[1-9][0-9]*)$"
			if t.Kind() == reflect.Int64 {
				pattern = "^(0|-?[1-9][0-9]*)$"
			}
			return map[string]any{"type": "string", "pattern": pattern, "maxLength": 20}
		}
		switch t.Kind() {
		case reflect.Struct:
			name := t.Name()
			if _, exists := defs[name]; !exists {
				defs[name] = nil
				properties := map[string]any{}
				required := []string{}
				for i := 0; i < t.NumField(); i++ {
					f := t.Field(i)
					tag := f.Tag.Get("json")
					if tag == "" || tag == "-" {
						continue
					}
					key := strings.Split(tag, ",")[0]
					p := schema(f.Type, strings.Contains(tag, ",string"))
					switch name + "." + f.Name {
					case "SaveV1.Format":
						p["const"] = "wwhsrt-save"
					case "SaveV1.SchemaVersion":
						p["const"] = 1
					case "SaveV1.CreatedAtUTC":
						p["format"] = "date-time"
						p["pattern"] = "Z$"
					case "SaveV1.PayloadSHA256", "DataRef.SHA256":
						p["pattern"] = "^[0-9a-f]{64}$"
					case "GameMonth.Year":
						p["minimum"] = 1
						p["maximum"] = 9999
					case "GameMonth.Month":
						p["minimum"] = 1
						p["maximum"] = 12
					case "WorldV1.CampaignMonths", "WorldV1.RestructuringWindowMonths":
						p["minimum"] = 1
					case "WorldV1.Companies":
						p["minItems"] = 1
						p["maxItems"] = 8
					case "Company.Status":
						p["enum"] = []string{"active", "restructuring", "bankrupt", "removed"}
					case "Ownership.Kind":
						p["enum"] = []string{"public", "company", "estate"}
					case "Node.Kind":
						p["enum"] = []string{"station", "junction"}
					case "Station.Status":
						p["enum"] = []string{"construction", "operational", "closed"}
					case "TrackSection.Access":
						p["enum"] = []string{"regulated_open", "owner_only"}
					case "TrackSection.Status":
						p["enum"] = []string{"construction", "operational", "closed"}
					case "Project.Kind":
						p["enum"] = []string{"new_line", "alignment_upgrade", "capacity_upgrade", "station_upgrade"}
					case "Project.Status":
						p["enum"] = []string{"pending", "committed", "building", "completed", "cancelled"}
					case "TrackAccessOffer.FeePerTrainKM", "TrackAccessAgreement.FeePerTrainKM":
						p["pattern"] = "^[1-9][0-9]*$"
					case "Trainset.Status":
						p["enum"] = []string{"delivery", "operational", "relocation", "retired"}
					case "TrainOrder.Status":
						p["enum"] = []string{"ordered", "delivered", "cancelled"}
					case "Auction.Status":
						p["enum"] = []string{"listed", "sold", "unsold", "withdrawn"}
					case "Reservation.Kind":
						p["enum"] = []string{"construction", "auction", "train_order"}
					}
					properties[key] = p
					required = append(required, key)
				}
				defs[name] = map[string]any{
					"type":                 "object",
					"additionalProperties": false,
					"properties":           properties,
					"required":             required,
				}
			}
			return map[string]any{"$ref": "#/$defs/" + name}
		case reflect.Slice:
			return map[string]any{"type": "array", "items": schema(t.Elem(), false)}
		case reflect.String:
			p := map[string]any{"type": "string"}
			if strings.HasSuffix(t.Name(), "ID") {
				p["pattern"] = "^[a-z][a-z0-9_-]{0,63}$"
			}
			return p
		case reflect.Bool:
			return map[string]any{"type": "boolean"}
		default:
			maximum := uint64(9007199254740991)
			if t.Bits() < 53 {
				maximum = (uint64(1) << t.Bits()) - 1
			}
			return map[string]any{"type": "integer", "minimum": 0, "maximum": maximum}
		}
	}
	root := schema(reflect.TypeFor[SaveV1](), false)
	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	root["title"] = "World-wide HSR portable save v1 (unreleased design contract)"
	root["$defs"] = defs
	return json.MarshalIndent(root, "", "  ")
}
