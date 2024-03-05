package log_test

import (
	"testing"

	"github.com/jimjibone/woodhouse-4/log"
)

func TestLogger(t *testing.T) {
	logger := log.NewLogger()
	logger.Debug("debug message %d %f 0x%x", 1, 2.3, 25)
	logger.Info("info message %d %f 0x%x", 1, 2.3, 25)
	logger.Warn("warn message %d %f 0x%x", 1, 2.3, 25)
	logger.Error("error message %d %f 0x%x", 1, 2.3, 25)
	// logger.Fatal("fatal message %d %f 0x%x", 1, 2.3, 25)
}
