package config

import "testing"

func TestFromEnvironment(t *testing.T) {
	t.Setenv("KZLOGVIEWER_NO_UPDATE_CHECK", "")
	if FromEnvironment().DisableUpdateCheck {
		t.Error("expected DisableUpdateCheck = false when unset")
	}

	t.Setenv("KZLOGVIEWER_NO_UPDATE_CHECK", "1")
	if !FromEnvironment().DisableUpdateCheck {
		t.Error("expected DisableUpdateCheck = true when set")
	}
}
