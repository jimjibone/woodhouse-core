package log

import (
	"fmt"

	"github.com/fatih/color"
)

type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return color.GreenString("DEBU")
	case InfoLevel:
		return color.BlueString("INFO")
	case WarnLevel:
		return color.YellowString("WARN")
	case ErrorLevel:
		return color.RedString("ERRO")
	case FatalLevel:
		return color.MagentaString("FATA")
	}
	return fmt.Sprintf("UNKNOWN(%d)", l)
}
