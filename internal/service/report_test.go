package service

import "testing"

func TestIKMFromRating(t *testing.T) {
	cases := []struct {
		name      string
		avg       float64
		count     int
		wantGrade string
	}{
		{"no data", 0, 0, "-"},
		{"perfect -> A", 5.0, 10, "A"},   // index 100
		{"4.5 -> A", 4.5, 4, "A"},        // index 90
		{"4.2 -> B", 4.2, 5, "B"},        // index 84
		{"3.5 -> C", 3.5, 6, "C"},        // index 70
		{"2.0 -> D", 2.0, 3, "D"},        // index 40
		{"boundary A 88.31", 4.4155, 2, "A"},
		{"boundary B low 76.61", 3.8305, 2, "B"},
		{"boundary C low 65.0", 3.25, 2, "C"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			idx, grade, label := ikmFromRating(c.avg, c.count)
			if grade != c.wantGrade {
				t.Errorf("avg=%v count=%d: got grade %q (idx %.2f), want %q", c.avg, c.count, grade, idx, c.wantGrade)
			}
			if c.count == 0 && idx != 0 {
				t.Errorf("no-data index should be 0, got %.2f", idx)
			}
			if label == "" {
				t.Errorf("label should never be empty")
			}
		})
	}
}
