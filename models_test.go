package mdk

import (
	"encoding/json"
	"testing"
)

func TestMetadataValueAndScan(t *testing.T) {
	t.Run("Scan valid JSON string", func(t *testing.T) {
		var m Metadata
		jsonStr := `{"key1":"value1","key2":123.45}`
		err := m.Scan(jsonStr)
		if err != nil {
			t.Fatalf("unexpected scan error: %v", err)
		}
		if m["key1"] != "value1" || m["key2"] != 123.45 {
			t.Errorf("unexpected scan output: %+v", m)
		}
	})

	t.Run("Scan valid []byte JSON", func(t *testing.T) {
		var m Metadata
		jsonBytes := []byte(`{"hello":"world"}`)
		err := m.Scan(jsonBytes)
		if err != nil {
			t.Fatalf("unexpected scan error: %v", err)
		}
		if m["hello"] != "world" {
			t.Errorf("expected world, got %v", m["hello"])
		}
	})

	t.Run("Scan nil value", func(t *testing.T) {
		var m Metadata = Metadata{"existing": "data"}
		err := m.Scan(nil)
		if err != nil {
			t.Fatalf("unexpected scan error: %v", err)
		}
		if m != nil {
			t.Errorf("expected m to be nil, got %+v", m)
		}
	})

	t.Run("Scan invalid type", func(t *testing.T) {
		var m Metadata
		err := m.Scan(123)
		if err == nil {
			t.Error("expected error for non-string/non-byte scan input")
		}
	})

	t.Run("Value serialization success", func(t *testing.T) {
		m := Metadata{"foo": "bar", "num": 1}
		val, err := m.Value()
		if err != nil {
			t.Fatalf("unexpected Value() error: %v", err)
		}
		valStr, ok := val.(string)
		if !ok {
			t.Fatalf("expected string driver value, got %T", val)
		}

		var parsed map[string]interface{}
		if err := json.Unmarshal([]byte(valStr), &parsed); err != nil {
			t.Fatalf("failed to parse Value output: %v", err)
		}
		if parsed["foo"] != "bar" || parsed["num"] != 1.0 {
			t.Errorf("unexpected parsed values: %+v", parsed)
		}
	})

	t.Run("Value serialization nil map", func(t *testing.T) {
		var m Metadata
		val, err := m.Value()
		if err != nil {
			t.Fatalf("unexpected Value() error: %v", err)
		}
		if val != nil {
			t.Errorf("expected nil value, got %v", val)
		}
	})
}
