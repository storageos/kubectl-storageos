package model

import "time"

// LogLevel is a typed wrapper around a cluster's log level configuration.
type LogLevel string

// LogLevelFromString wraps level as a LogLevel.
func LogLevelFromString(level string) LogLevel {
	return LogLevel(level)
}

// String returns the string representation of l.
func (l LogLevel) String() string {
	return string(l)
}

// LogFormat is a typed wrapper around a cluster's log entry format
// configuration.
type LogFormat string

// LogFormatFromString wraps format as a LogFormat.
func LogFormatFromString(format string) LogFormat {
	return LogFormat(format)
}

// String returns the string representation of f.
func (f LogFormat) String() string {
	return string(f)
}

// Resource encapsulate a StorageOS cluster api resource as a data type.
type Cluster struct {
	ID string `json:"id"`

	DisableTelemetry      bool `json:"disableTelemetry"`
	DisableCrashReporting bool `json:"disableCrashReporting"`
	DisableVersionCheck   bool `json:"disableVersionCheck"`

	LogLevel  LogLevel  `json:"logLevel"`
	LogFormat LogFormat `json:"logFormat"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Version   string    `json:"version"`
}
