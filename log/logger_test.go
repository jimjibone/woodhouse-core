package log_test

import (
	"testing"

	"github.com/jimjibone/woodhouse-4/log"
)

func TestLogger(t *testing.T) {
	logger := log.NewLogger(log.WithExitOnFatal(false), log.WithMinLevel(log.InfoLevel))

	logger.Debugf("debug format %d %f 0x%x", 1, 2.3, 25)
	logger.Infof("info format %d %f 0x%x", 1, 2.3, 25)
	logger.Warnf("warn format %d %f 0x%x", 1, 2.3, 25)
	logger.Errorf("error format %d %f 0x%x", 1, 2.3, 25)
	logger.Fatalf("fatal format %d %f 0x%x", 1, 2.3, 25)

	logger.Debugln("debug line", 1, 2.3, 25)
	logger.Infoln("info line", 1, 2.3, 25)
	logger.Warnln("warn line", 1, 2.3, 25)
	logger.Errorln("error line", 1, 2.3, 25)
	logger.Fatalln("fatal line", 1, 2.3, 25)
}
