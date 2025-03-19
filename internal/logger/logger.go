package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
)

const (
	InfoColor    = "\033[1;34m%s\033[0m"
	NoticeColor  = "\033[1;36m%s\033[0m"
	WarningColor = "\033[1;33m%s\033[0m"
	ErrorColor   = "\033[1;31m%s\033[0m"
	DebugColor   = "\033[0;36m%s\033[0m"
	ClearLine    = "\r\033[K"
	MoveUp       = "\033[1A"
	MoveDown     = "\033[1B"
)

var (
	bar *progressbar.ProgressBar
)

// StartProgress initializes a progress bar with the given description and total
func StartProgress(description string, total int) {
	// Clear any existing progress bar
	if bar != nil {
		bar.Finish()
		bar = nil
	}

	// Print a newline to create space for the progress bar
	fmt.Println()

	// Create new progress bar with custom options
	bar = progressbar.NewOptions(total,
		progressbar.OptionSetDescription(description),
		progressbar.OptionSetWidth(30),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
		progressbar.OptionOnCompletion(func() {
			fmt.Println()
		}),
		progressbar.OptionSetPredictTime(false),
		progressbar.OptionShowCount(),
		progressbar.OptionSetWriter(os.Stderr),
	)
}

// UpdateProgress updates the progress bar
func UpdateProgress() {
	if bar != nil {
		bar.Add(1)
	}
}

// FinishProgress completes the progress bar
func FinishProgress() {
	if bar != nil {
		bar.Finish()
		bar = nil
	}
}

// sleep adds a small delay between log messages for better readability
func sleep() {
	time.Sleep(100 * time.Millisecond)
}

func Info(format string, args ...interface{}) {
	sleep()
	// Move up one line to write above the progress bar
	fmt.Printf(MoveUp+ClearLine+InfoColor+"\n", fmt.Sprintf("ℹ️  "+format, args...))
}

func Success(format string, args ...interface{}) {
	sleep()
	fmt.Printf(MoveUp+ClearLine+NoticeColor+"\n", fmt.Sprintf("✅ "+format, args...))
}

func Warning(format string, args ...interface{}) {
	sleep()
	fmt.Printf(MoveUp+ClearLine+WarningColor+"\n", fmt.Sprintf("⚠️  "+format, args...))
}

func Error(format string, args ...interface{}) {
	sleep()
	fmt.Printf(MoveUp+ClearLine+ErrorColor+"\n", fmt.Sprintf("❌ "+format, args...))
}

func Debug(format string, args ...interface{}) {
	sleep()
	fmt.Printf(MoveUp+ClearLine+DebugColor+"\n", fmt.Sprintf("🔍 "+format, args...))
}

func Section(name string) {
	sleep()
	fmt.Printf(MoveUp+ClearLine+"\n%s\n%s", strings.Repeat("=", len(name)+4), name)
}

// StartSpinner starts a loading spinner with the given message
func StartSpinner(message string) *progressbar.ProgressBar {
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription(message),
		progressbar.OptionSetWidth(10),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	return bar
}
