package log

import (
	"testing"

	"fmt"
	"log/slog"
	"bytes"
	"encoding/json"
	"regexp"
)

type setupType struct {
	logger *RelayLogger
	buffer bytes.Buffer
}

func beforeEach(t *testing.T) *setupType {
	var r setupType

	err := InitLoggerWithWriter("info", "json", &r.buffer, false)
	if err != nil {
		t.Fatal(err)
	}

	r.logger = GetLogger()

	return &r
}

type logType struct {
	Time string
	Level string
	Source struct {
		Function string
		File string
		Line int
	}
	Msg string
	Stack string
	Error string
}

func parseResult(setup *setupType, t *testing.T) (string, logType) {
	raw := setup.buffer.String()
	var parsed logType

	err := json.Unmarshal(setup.buffer.Bytes(), &parsed)
	if err != nil {
		t.Fatalf("fail to parse log: %v: %s", err, raw)
	}

	return raw, parsed
}

func TestLogLevel(t *testing.T) {
	setup := beforeEach(t)

	setup.logger.log(slog.LevelDebug, 0, "test")
	if 0 < setup.buffer.Len() {
		t.Fatalf("debug log is output: %s", setup.buffer.String())
	}
}

func TestLogLog(t *testing.T) {
	setup := beforeEach(t)

	setup.logger.log(slog.LevelInfo, 0, "test")
	raw, r := parseResult(setup, t)

	if r.Level != "INFO" {
		t.Fatalf("mismatch level: %s", raw)
	}

	if m, err := regexp.MatchString(`/log.TestLogLog$`, r.Source.Function); err != nil || !m {
		t.Fatalf("mismatch source.function: %v", raw)
	}
}

func TestLogError(t *testing.T) {
	setup := beforeEach(t)

	setup.logger.Error("testerr", fmt.Errorf("dummy"))
	raw, r := parseResult(setup, t)

	if r.Level != "ERROR" {
		t.Fatalf("mismatch level: %s", raw)
	}

	if m, err := regexp.MatchString(`/log.TestLogError$`, r.Source.Function); err != nil || !m {
		t.Fatalf("mismatch source.function: %v", raw)
	}

	if r.Error != "dummy" {
		t.Fatalf("mismatch level: %s", raw)
	}
}
