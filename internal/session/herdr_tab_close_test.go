package session

import (
	"slices"
	"testing"
	"time"
)

func herdrTabNumbers(t *testing.T, sp, wsID string) []int {
	t.Helper()
	tabs := herdrOK(t, "tab.list", herdrDial(t, sp, "tab.list", map[string]any{"workspace_id": wsID}))["tabs"].([]any)
	var out []int
	for _, x := range tabs {
		out = append(out, int(x.(map[string]any)["number"].(float64)))
	}
	slices.Sort(out)
	return out
}

func herdrWaitTabs(t *testing.T, sp, wsID string, want []int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var got []int
	for time.Now().Before(deadline) {
		if got = herdrTabNumbers(t, sp, wsID); slices.Equal(got, want) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("tabs %v, want %v", got, want)
}

func herdrTabFixture(t *testing.T) (d *Daemon, sp, wsID string, tab map[string]any) {
	t.Helper()
	d, sp = startTestDaemon(t)
	ws := herdrOK(t, "workspace.create", herdrDial(t, sp, "workspace.create", map[string]any{"cwd": t.TempDir(), "focus": false, "label": "Demo"}))
	wsID = ws["workspace"].(map[string]any)["workspace_id"].(string)
	tab = herdrOK(t, "tab.create", herdrDial(t, sp, "tab.create", map[string]any{"workspace_id": wsID, "focus": false, "label": "task"}))
	return d, sp, wsID, tab
}

func TestHerdrPaneCloseOfTheLastPaneClosesTheTab(t *testing.T) {
	_, sp, wsID, tab := herdrTabFixture(t)
	herdrWaitTabs(t, sp, wsID, []int{1, 2})
	pane := tab["root_pane"].(map[string]any)["pane_id"].(string)
	herdrOK(t, "pane.close", herdrDial(t, sp, "pane.close", map[string]any{"pane_id": pane}))
	herdrWaitTabs(t, sp, wsID, []int{1})
	again := herdrOK(t, "tab.create", herdrDial(t, sp, "tab.create", map[string]any{"workspace_id": wsID, "focus": false}))
	if got := again["tab"].(map[string]any)["label"]; got != "2" {
		t.Errorf("tab.create took workspace %v, want the freed 2", got)
	}
}

func TestHerdrPaneExitOfTheLastPaneClosesTheTab(t *testing.T) {
	_, sp, wsID, tab := herdrTabFixture(t)
	pane := tab["root_pane"].(map[string]any)["pane_id"].(string)
	herdrOK(t, "send_text", herdrDial(t, sp, "pane.send_text", map[string]any{"pane_id": pane, "text": "exit"}))
	herdrOK(t, "send_keys", herdrDial(t, sp, "pane.send_keys", map[string]any{"pane_id": pane, "keys": []string{"Enter"}}))
	herdrWaitTabs(t, sp, wsID, []int{1})
}

func TestHerdrTabStaysWhileItHoldsAPane(t *testing.T) {
	_, sp, wsID, tab := herdrTabFixture(t)
	pane := tab["root_pane"].(map[string]any)["pane_id"].(string)
	split := herdrOK(t, "pane.split", herdrDial(t, sp, "pane.split", map[string]any{"target_pane_id": pane, "direction": "right"}))
	herdrOK(t, "pane.close", herdrDial(t, sp, "pane.close", map[string]any{"pane_id": pane}))
	time.Sleep(300 * time.Millisecond)
	if got := herdrTabNumbers(t, sp, wsID); !slices.Equal(got, []int{1, 2}) {
		t.Fatalf("tabs %v after one of two panes closed, want [1 2]", got)
	}
	herdrOK(t, "pane.close", herdrDial(t, sp, "pane.close", map[string]any{"pane_id": split["pane"].(map[string]any)["pane_id"]}))
	herdrWaitTabs(t, sp, wsID, []int{1})
}

func TestHerdrLastTabOfASessionStays(t *testing.T) {
	_, sp, wsID, _ := herdrTabFixture(t)
	first := herdrOK(t, "pane.list", herdrDial(t, sp, "pane.list", map[string]any{"workspace_id": wsID}))["panes"].([]any)
	var root string
	for _, p := range first {
		if p.(map[string]any)["tab_id"] == wsID+":t1" {
			root = p.(map[string]any)["pane_id"].(string)
		}
	}
	herdrOK(t, "pane.close", herdrDial(t, sp, "pane.close", map[string]any{"pane_id": root}))
	herdrWaitTabs(t, sp, wsID, []int{2})
	only := herdrOK(t, "pane.list", herdrDial(t, sp, "pane.list", map[string]any{"workspace_id": wsID}))["panes"].([]any)
	herdrOK(t, "pane.close", herdrDial(t, sp, "pane.close", map[string]any{"pane_id": only[0].(map[string]any)["pane_id"]}))
	time.Sleep(300 * time.Millisecond)
	if got := herdrTabNumbers(t, sp, wsID); len(got) != 1 {
		t.Fatalf("tabs %v after the last pane closed, want the session's last tab kept", got)
	}
}
