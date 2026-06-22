package entity

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONMetadata json.RawMessage

func StepMeta(step string) JSONMetadata {
	b, err := json.Marshal(struct {
		Step string `json:"step"`
	}{Step: step})
	if err != nil {
		panic(err)
	}
	return JSONMetadata(b)
}

func ProviderMeta(provider string) JSONMetadata {
	b, err := json.Marshal(struct {
		Provider string `json:"provider"`
	}{Provider: provider})
	if err != nil {
		panic(err)
	}
	return JSONMetadata(b)
}

func (m JSONMetadata) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return []byte(m), nil
}

func (m *JSONMetadata) Scan(value interface{}) error {
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
	cp := make([]byte, len(raw))
	copy(cp, raw)
	*m = JSONMetadata(cp)
	return nil
}
