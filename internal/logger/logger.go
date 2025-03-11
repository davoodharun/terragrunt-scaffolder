package logger

import (
	"fmt"
	"strings"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
)

func Info(format string, args ...interface{}) {
	fmt.Printf(InfoColor+"\n", fmt.Sprintf("ℹ️  "+format, args...))
}

func Success(format string, args ...interface{}) {
	fmt.Printf(NoticeColor+"\n", fmt.Sprintf("✅ "+format, args...))
}

func Warning(format string, args ...interface{}) {
	fmt.Printf(WarningColor+"\n", fmt.Sprintf("⚠️  "+format, args...))
}

func Error(format string, args ...interface{}) {
	fmt.Printf(ErrorColor+"\n", fmt.Sprintf("❌ "+format, args...))
}

func Debug(format string, args ...interface{}) {
	fmt.Printf(DebugColor+"\n", fmt.Sprintf("🔍 "+format, args...))
}

func Section(name string) {
	fmt.Printf("\n%s\n%s\n", strings.Repeat("=", len(name)+4), name)
}
