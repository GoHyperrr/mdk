package mdk

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// Metadata represents custom optional JSON metadata stored as a text/json field.
type Metadata map[string]interface{}

// Value returns the driver Value.
func (m Metadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	ba, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(ba), nil
}

// Scan scans value into Metadata.
func (m *Metadata) Scan(val interface{}) error {
	if val == nil {
		*m = nil
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New("failed to scan Metadata: invalid type")
	}
	t := make(map[string]interface{})
	if err := json.Unmarshal(ba, &t); err != nil {
		return err
	}
	*m = t
	return nil
}
