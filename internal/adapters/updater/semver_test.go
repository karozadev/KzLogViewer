package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.1", "v1.0.0", false},
		{"v1.0.0", "v1.0.0", false},
		{"v1.0.0", "v2.0.0", true},
		{"v1.2.0", "v1.10.0", true},
		{"v1.0.0-beta.1", "v1.0.0-beta.2", true},
		{"v1.0.0-beta.2", "v1.0.0-beta.1", false},
		{"v1.0.0-beta.1", "v1.0.0", true},
		{"v1.0.0", "v1.0.0-beta.1", false},
		{"dev", "v1.0.0", true},
	}
	for _, c := range cases {
		if got := IsNewer(c.current, c.latest); got != c.want {
			t.Errorf("IsNewer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}
