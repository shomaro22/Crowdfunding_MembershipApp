package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONMetadata map[string]any

func (m JSONMetadata) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (m *JSONMetadata) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}
	var raw []byte
	switch v := value.(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return fmt.Errorf("entity.JSONMetadata: unsupported type %T", value)
	}
	if len(raw) == 0 {
		*m = nil
		return nil
	}
	return json.Unmarshal(raw, m)
}
