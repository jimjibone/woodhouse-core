package api

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Frame struct {
	Topic   string          `json:"topic"`
	Payload json.RawMessage `json:"payload"`
}

func (frame *Frame) String() string {
	return fmt.Sprintf("topic: %q, payload: %s", frame.Topic, frame.Payload)
}

func SaveJSON(filename string, data []byte) {
	var tmp interface{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		panic(err)
	}
	payload, err := json.MarshalIndent(tmp, "", "    ")
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(filename, payload, 0644)
	if err != nil {
		panic(err)
	}
}

func SanitiseJSON(raw []byte) []byte {
	payload := strings.ReplaceAll(string(raw), `\"`, `"`)
	payload = strings.TrimPrefix(payload, `"`)
	payload = strings.TrimSuffix(payload, `"`)
	return json.RawMessage(payload)
}
