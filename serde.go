package lrucache

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func serialize(value interface{}) ([]byte, error) {
	switch v := value.(type) {
	case string:
		return []byte(v), nil
	case []byte:
		return v, nil
	case int:
		return []byte(fmt.Sprintf("%d", v)), nil
	case int32:
		return []byte(fmt.Sprintf("%d", v)), nil
	case int64:
		return []byte(fmt.Sprintf("%d", v)), nil
	case float32:
		return []byte(fmt.Sprintf("%f", v)), nil
	case float64:
		return []byte(fmt.Sprintf("%f", v)), nil
	case bool:
		return []byte(fmt.Sprintf("%t", v)), nil
	default:
		return json.Marshal(v)
	}
}

func deserialize(data []byte) (interface{}, error) {
	// Try to parse as int
	if i, err := strconv.Atoi(string(data)); err == nil {
		return i, nil
	}

	// Try to parse as float
	if f, err := strconv.ParseFloat(string(data), 64); err == nil {
		return f, nil
	}

	// Try to parse as bool
	if b, err := strconv.ParseBool(string(data)); err == nil {
		return b, nil
	}

	// If not a simple type, try JSON
	var value interface{}
	if err := json.Unmarshal(data, &value); err == nil {
		return value, nil
	}

	// Treat as string
	return string(data), nil
}
