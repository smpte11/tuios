package session

// herdr closes a tab when its last pane closes. A tuios workspace is a fixed
// slot that outlives its panes, so the herdr front reaps the slots it handed
// out itself: a workspace opened by herdr's tab.create or pane.move is a herdr
// tab, and when its last pane goes the daemon takes its label off, shows
// another tab if it was the one showing, and so frees the slot for the next
// tab.create. A workspace a person made or named in tuios is never touched.
//
// herdr closes the workspace when its last tab empties. tuios does not end a
// session because a pane closed, so the last tab of a session stays, empty,
// as an emptied session does in tuios.

// MarkHerdrTab records that workspace ws is a herdr tab, which closes with
// its last pane.
func (s *Session) MarkHerdrTab(ws int) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if ws < 1 || IsScratchWorkspace(ws) {
		return
	}
	if s.herdrTabs == nil {
		s.herdrTabs = make(map[int]struct{})
	}
	s.herdrTabs[ws] = struct{}{}
}

// herdrTabsEmptyLocked reports whether a herdr tab has lost its last pane
// while another tab remains. The caller holds stateMu.
func (s *Session) herdrTabsEmptyLocked(state *SessionState) bool {
	for ws := range s.herdrTabs {
		if herdrCount(state, ws) == 0 && herdrOtherTab(state, ws) != 0 {
			return true
		}
	}
	return false
}

// herdrOtherTab is the tab to show in place of the emptied ws: the nearest
// one before it in display order that holds a pane, else the nearest after
// it, else any other listed tab, else 0.
func herdrOtherTab(state *SessionState, ws int) int {
	if next := herdrNeighbourTab(state, ws); next != 0 {
		return next
	}
	for _, n := range herdrWorkspaceOrder(state) {
		if n != ws && herdrListedWorkspace(state, n) {
			return n
		}
	}
	return 0
}

// reapHerdrTabsLocked closes the herdr tabs that hold no pane, but never a
// session's last tab. The caller holds stateMu, inside mutateState.
func (s *Session) reapHerdrTabsLocked(state *SessionState) {
	for ws := range s.herdrTabs {
		if herdrCount(state, ws) > 0 {
			continue
		}
		next := herdrOtherTab(state, ws)
		if next == 0 {
			continue
		}
		delete(s.herdrTabs, ws)
		delete(state.WorkspaceNames, ws)
		delete(state.WorkspaceFocus, ws)
		if state.CurrentWorkspace == ws {
			s.markFocusIntentLocked()
			state.CurrentWorkspace = next
			focus := state.WorkspaceFocus[next]
			if focus == "" {
				focus = firstVisibleOnWorkspace(state.Windows, next)
			}
			state.FocusedWindowID = focus
		}
	}
}

// reapHerdrTabsSoon reaps after a client's push closed a pane, by a mutation
// of the daemon's own, so the push is never rewritten under the client.
func (s *Session) reapHerdrTabsSoon() {
	go func() { _ = s.mutateState(func(*SessionState) error { return nil }) }()
}

// inHerdrTab reports whether window id is a pane of a herdr tab. herdr closes
// a pane whose process exits, and nothing closes it for a session with no
// client attached, so the daemon does for the panes of herdr tabs.
func (s *Session) inHerdrTab(id string) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if len(s.herdrTabs) == 0 || s.state == nil {
		return false
	}
	for i := range s.state.Windows {
		w := &s.state.Windows[i]
		if w.ID != id || herdrScratch(w) {
			continue
		}
		_, ok := s.herdrTabs[w.Workspace]
		return ok
	}
	return false
}
