package tui

import "testing"

func TestNewModelDefaults(t *testing.T) {
	m := newTestModel()
	if m.maxEntries != defaultMaxEntries {
		t.Errorf("maxEntries = %d, want %d", m.maxEntries, defaultMaxEntries)
	}
	if !m.autoscroll {
		t.Error("expected autoscroll = true by default")
	}
	if m.mode != viewList {
		t.Error("expected initial mode = viewList")
	}
}

func TestInitReturnsBatchedCommand(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected a non-nil init command")
	}
}
