package connie

import "encoding/json"

func schema(obj map[string]any) json.RawMessage {
	b, err := json.Marshal(obj)
	if err != nil {
		return json.RawMessage(`{"type":"object","properties":{}}`)
	}
	return b
}

func emptyObjectSchema() json.RawMessage {
	return schema(map[string]any{
		"type":       "object",
		"properties": map[string]any{},
		"additionalProperties": false,
	})
}

func propString(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func propInt(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}
