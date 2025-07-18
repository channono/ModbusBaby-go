package logger

import (
	"bytes"
	"fmt"

	"github.com/sirupsen/logrus"
)

// CustomFormatter implements logrus.Formatter interface to provide custom log format.
type CustomFormatter struct{}

// Format renders a single log entry
func (f *CustomFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Format timestamp
	timestamp := entry.Time.Format("2006-01-02T15:04:05.000-07:00") // Matches original timestamp format

	// Write the formatted log entry
	b.WriteString(fmt.Sprintf("%s | %s\n", timestamp, entry.Message))

	return b.Bytes(), nil
}
