package logger

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/manifoldco/promptui"
)

// Logger is the cli's logger.Logger implementation
type Logger struct {
	Writer        io.Writer
	writerMu      sync.Mutex
	smartTerminal bool
	Verbose       bool
}

func NewLogger() *Logger {
	return &Logger{
		Writer:        os.Stdout,
		smartTerminal: IsSmartTerminal(os.Stdout),
		Verbose:       false,
	}
}

func (l *Logger) Prompt(message string) {
	l.println(l.formatPrompt(message))
}

func (l *Logger) Info(message string) {
	if l.Verbose {
		l.println(message)
	}
}

func (l *Logger) Infof(message string, args ...interface{}) {
	if l.Verbose {
		l.println(message, args...)
	}
}

func (l *Logger) Warn(message string) {
	l.println(l.formatWithIcon(promptui.IconWarn, message))
}

func (l *Logger) Warnf(message string, args ...interface{}) {
	l.println(l.formatWithIcon(promptui.IconWarn, message), args...)
}

func (l *Logger) Error(message string) {
	l.println(l.formatWithIcon(promptui.IconBad, message))
}

func (l *Logger) Errorf(message string, args ...interface{}) {
	l.println(l.formatWithIcon(promptui.IconBad, message), args...)
}

func (l *Logger) Success(message string) {
	l.println(l.formatWithIcon(promptui.IconGood, message))
}

func (l *Logger) Successf(message string, args ...interface{}) {
	l.println(l.formatWithIcon(promptui.IconGood, message), args...)
}

func (l *Logger) println(message string, args ...interface{}) {
	l.writerMu.Lock()
	defer l.writerMu.Unlock()
	fmt.Fprintln(l.Writer, fmt.Sprintf(message, args...))
}

func (l *Logger) formatPrompt(message string) string {
	if l.smartTerminal {
		message = promptui.Styler(promptui.FGBold)(message)
	}

	return fmt.Sprintf("%s%s", message, "\n")
}

func (l *Logger) formatWithIcon(icon, message string) string {
	if l.smartTerminal {
		icon = promptui.Styler(promptui.FGBold)(icon)
		message = promptui.Styler(promptui.FGBold)(message)
		message = fmt.Sprintf("%s %s", icon, message)
	}

	return message
}

func (l *Logger) Commencing(command string) {
	commencingMessage := fmt.Sprintf("Commencing %s, this may take a few moments.", command)
	if l.smartTerminal {
		timer := ("⏳")
		commencingMessage = promptui.Styler(promptui.FGBold)(commencingMessage)
		commencingMessage = fmt.Sprintf("%s%s", timer, commencingMessage)
	}
	l.println(commencingMessage)
}
