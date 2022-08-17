package log

import (
	"bytes"
	"github.com/sirupsen/logrus"
	"os"
	"strings"
	"text/template"
)

const (
	defaultTimestampFormat = "2006-01-02 15:04:05.000"
	defaultFormat          = "{{.timestamp}} {{.pid}} [{{.name}}] [{{.levelname}}] {{.message}}"
)

type LogFormatter struct {
	TimestampFormat string
	OutputFormat    string
}

func NewLogFormatter() *LogFormatter {
	return &LogFormatter{
		TimestampFormat: defaultTimestampFormat,
		OutputFormat:    defaultFormat,
	}
}

func (f *LogFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	data := map[string]interface{}{
		"timestamp": entry.Time.Format(f.TimestampFormat),
		"pid":       os.Getpid(),
		"levelname": strings.ToUpper(entry.Level.String()),
		"message":   entry.Message,
	}
	for key, value := range entry.Data {
		data[key] = value
	}

	t := template.Must(template.New("").Parse(f.OutputFormat))
	err := t.Execute(b, data)
	if err != nil {
		return nil, err
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
