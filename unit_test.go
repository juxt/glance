package main

import (
	"testing"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		secs int64
		want string
	}{
		{0, "unknown"},
		{59, "59s ago"},
		{60, "1m ago"},
		{3599, "59m ago"},
		{3600, "1h ago"},
		{86399, "23h ago"},
		{86400, "1d ago"},
	}
	for _, tt := range tests {
		got := formatAge(tt.secs)
		if got != tt.want {
			t.Errorf("formatAge(%d) = %q, want %q", tt.secs, got, tt.want)
		}
	}
}

func TestPluralLines(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{1, "1 line"},
		{0, "0 lines"},
		{5, "5 lines"},
	}
	for _, tt := range tests {
		got := pluralLines(tt.n)
		if got != tt.want {
			t.Errorf("pluralLines(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}

func TestSectionRanges(t *testing.T) {
	tests := []struct {
		name string
		nums []int
		want string
	}{
		{"empty", nil, ""},
		{"single", []int{5}, "5"},
		{"range", []int{1, 2, 3}, "1-3"},
		{"mixed", []int{1, 2, 3, 10, 20, 21, 22}, "1-3, 10, 20-22"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sectionRanges(tt.nums)
			if got != tt.want {
				t.Errorf("sectionRanges(%v) = %q, want %q", tt.nums, got, tt.want)
			}
		})
	}
}

func TestParsePresetLine(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		want    preset
		wantErr bool
	}{
		{"valid full", "/mypreset/error|fail/My description", preset{"mypreset", "error|fail", "My description"}, false},
		{"valid no desc", "/mypreset/error|fail", preset{"mypreset", "error|fail", ""}, false},
		{"comma delim", ",mypreset,a/b,Has slash", preset{"mypreset", "a/b", "Has slash"}, false},
		{"too short", "x", preset{}, true},
		{"empty", "", preset{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePresetLine(tt.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePresetLine(%q) error = %v, wantErr %v", tt.line, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parsePresetLine(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestIsValidPresetName(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"mypreset", true},
		{"my-preset", true},
		{"my_preset", true},
		{"UPPER", true},
		{"a1", true},
		{"", false},
		{"-dash", false},
		{"has space", false},
		{"has/slash", false},
	}
	for _, tt := range tests {
		got := isValidPresetName(tt.name)
		if got != tt.want {
			t.Errorf("isValidPresetName(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"1", 1},
		{"42", 42},
		{"0", 0},
		{"abc", 0},
		{"", 0},
		{"-1", 0},
	}
	for _, tt := range tests {
		got := parsePositiveInt(tt.s)
		if got != tt.want {
			t.Errorf("parsePositiveInt(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}
