package repository

import "testing"

func TestDeltaPct(t *testing.T) {
	cases := []struct {
		name        string
		today, yest int
		want        float64
	}{
		{"growth", 10, 5, 100},
		{"shrink", 5, 10, -50},
		{"both zero", 0, 0, 0},
		{"from zero", 5, 0, 100},
		{"flat", 7, 7, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := deltaPct(c.today, c.yest)
			if got != c.want {
				t.Errorf("deltaPct(%d, %d) = %v, want %v", c.today, c.yest, got, c.want)
			}
		})
	}
}
