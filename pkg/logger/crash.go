package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"time"
)

// RecoverAndLog catches panics and writes them to crash-logs/crash.log
func RecoverAndLog(contextInfo string) {
	if r := recover(); r != nil {
		errStr := fmt.Sprintf("[%s] Panic in %s: %v\n%s\n", time.Now().Format(time.RFC3339), contextInfo, r, string(debug.Stack()))
		log.Print(errStr)

		logDir := "crash-logs"
		if err := os.MkdirAll(logDir, 0755); err != nil {
			log.Printf("Failed to create crash-logs directory: %v", err)
			return
		}

		logFile := filepath.Join(logDir, "crash.log")
		if f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
			f.WriteString(errStr + "\n")
			f.Close()
		} else {
			log.Printf("Failed to write to crash.log: %v", err)
		}
	}
}
