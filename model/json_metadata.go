package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type JSONMetadata json.RawMessage

func StepMeta(step string) JSONMetadata {
	var (
		b   []byte
		err error
	)
	b, err = json.Marshal(struct {
		Step string `json:"step"`
	}{Step: step})
	if err != nil {
		panic(err)
	}
	return JSONMetadata(b)
}

func ProviderMeta(provider string) JSONMetadata {
	var (
		b   []byte
		err error
	)
	b, err = json.Marshal(struct {
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
	var (
		raw []byte
		cp  []byte
	)
	if value == nil {
		*m = nil
		return nil
	}
	switch value.(type) {
	case []byte:
		raw = value.([]byte)
	case string:
		raw = []byte(value.(string))
	default:
		return fmt.Errorf("model.JSONMetadata: unsupported type %T", value)
	}
	if len(raw) == 0 {
		*m = nil
		return nil
	}
	cp = make([]byte, len(raw))
	copy(cp, raw)
	*m = JSONMetadata(cp)
	return nil
}
