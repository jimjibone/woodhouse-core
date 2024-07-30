package api

import (
	"fmt"
)

type Config struct {
	Cameras map[string]interface{} `json:"cameras"`
}

func (config Config) String() string {
	return fmt.Sprintf("cameras: %s", config.Cameras)
}
