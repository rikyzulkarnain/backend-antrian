package domain

import "testing"

func TestServicePrefix_KnownTypes(t *testing.T) {
	cases := map[string]string{
		"UMUM": "A",
		"LAB":  "B",
		"AMP":  "C",
		"UTIL": "D",
		"SEWA": "E",
	}
	for in, want := range cases {
		got, ok := ServicePrefix(in)
		if !ok {
			t.Errorf("ServicePrefix(%q): expected ok=true", in)
			continue
		}
		if got != want {
			t.Errorf("ServicePrefix(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestServicePrefix_UnknownReturnsFalse(t *testing.T) {
	for _, bad := range []string{"", "umum", "FOO", "UMUMX", "umum "} {
		if _, ok := ServicePrefix(bad); ok {
			t.Errorf("ServicePrefix(%q): expected ok=false", bad)
		}
	}
}
