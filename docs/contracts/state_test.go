package contracts

import (
	"bytes"
	"flag"
	"os"
	"reflect"
	"strings"
	"testing"
)

var updateContracts = flag.Bool("update-contracts", false, "regenerate schema/example from reference contracts")

func fixture(t *testing.T) PayloadV1 {
	t.Helper()
	data, err := os.ReadFile("fixtures/minimal-payload.json")
	if err != nil {
		t.Fatal(err)
	}
	var p PayloadV1
	if err := decodeStrict(data, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSchemaAndExample(t *testing.T) {
	schema, err := JSONSchemaV1()
	if err != nil {
		t.Fatal(err)
	}
	save, err := EncodeV1(fixture(t), "2026-11-08T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string][]byte{
		"save-v1.schema.json":           append(schema, '\n'),
		"fixtures/minimal-save-v1.json": append(save, '\n'),
	} {
		if *updateContracts {
			if err := os.WriteFile(path, want, 0644); err != nil {
				t.Fatal(err)
			}
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Fatalf("%s is stale; regenerate explicitly", path)
		}
	}
	p, err := DecodeV1(save)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(p, fixture(t)) {
		t.Fatal("save round trip changed typed state")
	}
}

func TestLargeIntegerAndTamperBoundaries(t *testing.T) {
	p := fixture(t)
	p.World.Companies[0].Cash = Money(9007199254740993)
	p.World.Seed = ^uint64(0)
	encoded, err := EncodeV1(p, "2026-11-08T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"cash":"9007199254740993"`)) {
		t.Fatal("money lost decimal-string representation")
	}
	decoded, err := DecodeV1(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.World.Companies[0].Cash != p.World.Companies[0].Cash || decoded.World.Seed != p.World.Seed {
		t.Fatal("64-bit precision lost")
	}
	if _, err := DecodeV1(
		bytes.Replace(encoded, []byte("9007199254740993"), []byte("9007199254740994"), 1),
	); err == nil {
		t.Fatal("accepted modified payload")
	}
	if _, err := DecodeV1(
		bytes.Replace(encoded, []byte(`"schema_version":1`), []byte(`"schema_version":2`), 1),
	); err == nil {
		t.Fatal("accepted future schema")
	}
	if _, err := DecodeV1(
		bytes.Replace(
			encoded,
			[]byte(`"format":"wwhsrt-save"`),
			[]byte(`"format":"wwhsrt-save","format":"wwhsrt-save"`),
			1,
		),
	); err == nil {
		t.Fatal("accepted duplicate field")
	}
	if _, err := DecodeV1(append(encoded, []byte("{}")...)); err == nil {
		t.Fatal("accepted trailing document")
	}
}

func TestAllWideScalarsAreQuoted(t *testing.T) {
	seen := map[reflect.Type]bool{}
	var check func(reflect.Type)
	check = func(v reflect.Type) {
		if v.Kind() == reflect.Pointer || v.Kind() == reflect.Slice {
			check(v.Elem())
			return
		}
		if v.Kind() != reflect.Struct || seen[v] {
			return
		}
		seen[v] = true
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			fieldType := f.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
			if (fieldType.Kind() == reflect.Int64 || fieldType.Kind() == reflect.Uint64) &&
				!strings.Contains(f.Tag.Get("json"), ",string") {
				t.Errorf("%s.%s must encode as a decimal string", v.Name(), f.Name)
			}
			check(f.Type)
		}
	}
	check(reflect.TypeFor[PayloadV1]())
}
