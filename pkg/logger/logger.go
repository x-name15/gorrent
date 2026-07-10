package logger

import (
	"log"
)

// Level determines the verbosity of the application.
// Defaults to "minimal". Set to "debug" for verbose output.
var Level = "minimal"

// Debugf prints a formatted message to the standard logger only if Level is "debug".
func Debugf(format string, v ...interface{}) {
	if Level == "debug" {
		log.Printf("[DEBUG] "+format, v...)
	}
}
