package tui

import "testing"

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		s, pattern string
		want       bool
	}{
		{"hello world", "hlo", true},
		{"hello world", "hw", true},
		{"hello", "hello", true},
		{"hello", "hx", false},
		{"", "", true},
		{"abc", "", true},
		{"", "a", false},
		{"abcdef", "ace", true},
		{"abcdef", "aec", false},
	}

	for _, tt := range tests {
		got := fuzzyMatch(tt.s, tt.pattern)
		if got != tt.want {
			t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
		}
	}
}

func TestNewFilterModel(t *testing.T) {
	rows := [][]string{
		{"alice", "30"},
		{"bob", "25"},
		{"alice", "31"},
	}

	fm := newFilterModel(0, "name", rows, nil)

	if len(fm.allVals) != 2 {
		t.Fatalf("expected 2 unique values, got %d", len(fm.allVals))
	}
	// All should be selected by default
	for _, v := range fm.allVals {
		if !fm.selected[v] {
			t.Errorf("expected %q to be selected", v)
		}
	}
}

func TestNewFilterModelWithExisting(t *testing.T) {
	rows := [][]string{
		{"alice", "30"},
		{"bob", "25"},
	}

	existing := map[string]bool{"alice": true, "bob": false}
	fm := newFilterModel(0, "name", rows, existing)

	if fm.selected["alice"] != true {
		t.Error("expected alice to be selected")
	}
	if fm.selected["bob"] != false {
		t.Error("expected bob to be unselected")
	}
}
