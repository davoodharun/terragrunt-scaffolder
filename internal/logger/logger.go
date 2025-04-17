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
	MoveUp       = "\033[1A"
	MoveDown     = "\033[1B"
	ClearLine    = "\r\033[K"
)

var (
	bar     *progressbar.ProgressBar
	history []string
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

// shouldKeepInHistory determines if a success message should be kept in history
func shouldKeepInHistory(message string) bool {
	// Keep important success messages
	importantMessages := []string{
		"Infrastructure folder created",
		"Components created",
		"All components validated successfully",
	}

	for _, important := range importantMessages {
		if strings.Contains(message, important) {
			return true
		}
	}
	return false
}

// printWithHistory prints a message and maintains the history
func printWithHistory(message string, isSuccess bool) {
	if bar != nil {
		if isSuccess && shouldKeepInHistory(message) {
			// Check if message already exists in history
			exists := false
			for _, msg := range history {
				if msg == message {
					exists = true
					break
				}
			}
			// Only add message to history if it doesn't already exist
			if !exists {
				history = append(history, message)
			}
		}

		// Move up enough lines to show all history
		for i := 0; i < len(history); i++ {
			fmt.Print(MoveUp + ClearLine)
		}

		// Print all history
		for _, msg := range history {
			fmt.Println(msg)
		}

		// Print the current message
		fmt.Println(message)

		// Move back down to the progress bar
		for i := 0; i < len(history)+1; i++ {
			fmt.Print(MoveDown)
		}
	} else {
		fmt.Println(message)
	}
}

func Info(format string, args ...interface{}) {
	sleep()
	message := fmt.Sprintf(InfoColor, fmt.Sprintf("ℹ️  "+format, args...))
	printWithHistory(message, false)
}

func Success(format string, args ...interface{}) {
	sleep()
	message := fmt.Sprintf(NoticeColor, fmt.Sprintf("✅ "+format, args...))
	printWithHistory(message, true)
}

func Warning(format string, args ...interface{}) {
	sleep()
	message := fmt.Sprintf(WarningColor, fmt.Sprintf("⚠️  "+format, args...))
	printWithHistory(message, false)
}

func Error(format string, args ...interface{}) {
	sleep()
	message := fmt.Sprintf(ErrorColor, fmt.Sprintf("❌ "+format, args...))
	printWithHistory(message, false)
}

func Debug(format string, args ...interface{}) {
	sleep()
	message := fmt.Sprintf(DebugColor, fmt.Sprintf("🔍 "+format, args...))
	printWithHistory(message, false)
}

func Section(name string) {
	sleep()
	message := fmt.Sprintf("\n%s\n%s", strings.Repeat("=", len(name)+4), name)
	printWithHistory(message, false)
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
