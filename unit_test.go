package main

import (
	"reflect"
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

func TestParsePipeArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want pipeConfig
	}{
		{"defaults", nil, pipeConfig{n: defaultHeadTail}},
		{"custom n", []string{"-n", "5"}, pipeConfig{n: 5}},
		{"long head", []string{"--head", "3"}, pipeConfig{n: 3}},
		{"long lines", []string{"--lines", "7"}, pipeConfig{n: 7}},
		{"no store", []string{"--no-store"}, pipeConfig{n: defaultHeadTail, noStore: true}},
		{"filter", []string{"-f", "error"}, pipeConfig{n: defaultHeadTail, filters: []string{"error"}}},
		{"multi filter", []string{"-f", "a", "-f", "b"}, pipeConfig{n: defaultHeadTail, filters: []string{"a", "b"}}},
		{"combined", []string{"-n", "3", "--no-store", "-f", "x"}, pipeConfig{n: 3, noStore: true, filters: []string{"x"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parsePipeArgs(tt.args)
			if err != nil {
				t.Fatalf("parsePipeArgs(%v) error: %v", tt.args, err)
			}
			if got.n != tt.want.n {
				t.Errorf("n = %d, want %d", got.n, tt.want.n)
			}
			if got.noStore != tt.want.noStore {
				t.Errorf("noStore = %v, want %v", got.noStore, tt.want.noStore)
			}
			if !reflect.DeepEqual(got.filters, tt.want.filters) {
				t.Errorf("filters = %v, want %v", got.filters, tt.want.filters)
			}
		})
	}
}

func TestParsePipeArgsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"missing n value", []string{"-n"}},
		{"invalid n", []string{"-n", "abc"}},
		{"zero n", []string{"-n", "0"}},
		{"unknown flag", []string{"--bogus"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePipeArgs(tt.args)
			if err == nil {
				t.Errorf("parsePipeArgs(%v) expected error", tt.args)
			}
		})
	}
}

func TestParseShowArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want showConfig
	}{
		{"id only", []string{"myid"}, showConfig{id: "myid"}},
		{"with lines", []string{"myid", "-l", "5-10"}, showConfig{id: "myid", ranges: [][2]int{{5, 10}}}},
		{"with filter", []string{"myid", "-f", "err"}, showConfig{id: "myid", filters: []string{"err"}}},
		{"with around", []string{"myid", "-a", "25", "3"}, showConfig{id: "myid", around: []aroundSpec{{center: 25, context: 3}}}},
		{"around default ctx", []string{"myid", "-a", "25"}, showConfig{id: "myid", around: []aroundSpec{{center: 25, context: defaultAroundContext}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseShowArgs(tt.args)
			if err != nil {
				t.Fatalf("parseShowArgs(%v) error: %v", tt.args, err)
			}
			if got.id != tt.want.id {
				t.Errorf("id = %q, want %q", got.id, tt.want.id)
			}
			if !reflect.DeepEqual(got.ranges, tt.want.ranges) {
				t.Errorf("ranges = %v, want %v", got.ranges, tt.want.ranges)
			}
			if !reflect.DeepEqual(got.filters, tt.want.filters) {
				t.Errorf("filters = %v, want %v", got.filters, tt.want.filters)
			}
			if !reflect.DeepEqual(got.around, tt.want.around) {
				t.Errorf("around = %v, want %v", got.around, tt.want.around)
			}
		})
	}
}

func TestParseShowArgsErrors(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{"no args", nil},
		{"invalid id slash", []string{"foo/bar"}},
		{"invalid id dotdot", []string{"foo..bar"}},
		{"invalid range", []string{"myid", "-l", "abc"}},
		{"unknown flag", []string{"myid", "--bogus"}},
		{"around bad center", []string{"myid", "-a", "abc"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseShowArgs(tt.args)
			if err == nil {
				t.Errorf("parseShowArgs(%v) expected error", tt.args)
			}
		})
	}
}

func TestRingBuffer(t *testing.T) {
	t.Run("under capacity", func(t *testing.T) {
		r := newRingBuffer(5)
		r.push(1, "a", false)
		r.push(2, "b", true)
		r.push(3, "c", false)
		got := r.entries()
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		want := []ringEntry{{1, "a", false}, {2, "b", true}, {3, "c", false}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("exact capacity", func(t *testing.T) {
		r := newRingBuffer(3)
		r.push(1, "a", false)
		r.push(2, "b", true)
		r.push(3, "c", false)
		got := r.entries()
		want := []ringEntry{{1, "a", false}, {2, "b", true}, {3, "c", false}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("evict unmatched", func(t *testing.T) {
		r := newRingBuffer(3)
		r.push(1, "a", false)
		r.push(2, "b", true)
		r.push(3, "c", false)
		evicted, ok := r.push(4, "d", false)
		if !ok {
			t.Fatal("expected eviction")
		}
		if evicted != (ringEntry{1, "a", false}) {
			t.Errorf("evicted = %+v, want {1 a false}", evicted)
		}
		got := r.entries()
		want := []ringEntry{{2, "b", true}, {3, "c", false}, {4, "d", false}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("evict matched", func(t *testing.T) {
		r := newRingBuffer(2)
		r.push(1, "ERROR", true)
		r.push(2, "ok", false)
		evicted, ok := r.push(3, "new", false)
		if !ok {
			t.Fatal("expected eviction")
		}
		if !evicted.matched {
			t.Errorf("evicted.matched = false, want true")
		}
		if evicted != (ringEntry{1, "ERROR", true}) {
			t.Errorf("evicted = %+v, want {1 ERROR true}", evicted)
		}
	})

	t.Run("wrap around twice", func(t *testing.T) {
		r := newRingBuffer(2)
		r.push(1, "a", true)
		r.push(2, "b", false)
		r.push(3, "c", false)
		r.push(4, "d", true)
		r.push(5, "e", false)
		got := r.entries()
		want := []ringEntry{{4, "d", true}, {5, "e", false}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("empty", func(t *testing.T) {
		r := newRingBuffer(5)
		got := r.entries()
		if len(got) != 0 {
			t.Errorf("empty ring: len = %d", len(got))
		}
	})

	t.Run("no eviction before full", func(t *testing.T) {
		r := newRingBuffer(3)
		_, ok := r.push(1, "a", false)
		if ok {
			t.Error("should not evict before full")
		}
		_, ok = r.push(2, "b", false)
		if ok {
			t.Error("should not evict before full")
		}
		_, ok = r.push(3, "c", false)
		if ok {
			t.Error("should not evict at exact capacity")
		}
		_, ok = r.push(4, "d", false)
		if !ok {
			t.Error("should evict after full")
		}
	})
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
