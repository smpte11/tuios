package herdrcli

import (
	"reflect"
	"strings"
	"testing"
)

func parseLine(t *testing.T, line string) (*Call, *UsageError) {
	t.Helper()
	return Parse(strings.Split(line, " "), func(string) string { return "" }, "/work")
}

func TestSessionFlagIsStrippedBeforeDashDash(t *testing.T) {
	want, werr := parseLine(t, "pane send-text w1:p1 hello")
	if werr != nil {
		t.Fatal(werr)
	}
	for _, line := range []string{
		"--session default pane send-text w1:p1 hello",
		"--session=default pane send-text w1:p1 hello",
		"pane --session default send-text w1:p1 hello",
		"pane send-text --session default w1:p1 hello",
		"pane send-text w1:p1 hello --session default",
		"pane send-text w1:p1 hello --session=default",
	} {
		got, err := parseLine(t, line)
		if err != nil {
			t.Errorf("%q: %v", line, err.Msg)
			continue
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q: %+v, want %+v", line, got, want)
		}
	}
}

func TestSessionFlagAfterDashDashIsKept(t *testing.T) {
	c, err := parseLine(t, "pane report-agent w1:p1 --source s --agent a --state working -- crush --session default")
	if err != nil {
		t.Fatal(err.Msg)
	}
	want := []string{"crush", "--session", "default"}
	if got, _ := c.Params["resume_argv"].([]string); !reflect.DeepEqual(got, want) {
		t.Errorf("resume_argv = %v, want %v", got, want)
	}
	c, err = parseLine(t, "pane report-agent w1:p1 --source s --agent a --state working -- crush --session=other")
	if err != nil {
		t.Fatal(err.Msg)
	}
	if got, _ := c.Params["resume_argv"].([]string); !reflect.DeepEqual(got, []string{"crush", "--session=other"}) {
		t.Errorf("resume_argv = %v", got)
	}
}

func TestSessionFlagNamesTheOneSession(t *testing.T) {
	for _, tc := range []struct{ line, msg string }{
		{"--session work pane list", `session "work" not found`},
		{"pane list --session=work", `session "work" not found`},
		{"pane list --session", "missing value for --session"},
		{"pane list --session=", "session name cannot be empty"},
		{"pane list --session bad/name", "session name may only contain"},
	} {
		c, err := parseLine(t, tc.line)
		if err == nil || c != nil {
			t.Errorf("%q: %+v, %v, want a usage error", tc.line, c, err)
			continue
		}
		if err.Code != 2 || !strings.Contains(err.Msg, tc.msg) {
			t.Errorf("%q: code %d %q, want %q", tc.line, err.Code, err.Msg, tc.msg)
		}
	}
}
