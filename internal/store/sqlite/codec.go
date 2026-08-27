package sqlite

import (
	"encoding/json"
	"fmt"
)

func encode(value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("序列化持久化对象: %w", err)
	}
	return data, nil
}

func decode(data []byte, target any) error {
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("反序列化持久化对象: %w", err)
	}
	return nil
}
