package source

import "testing"

func TestLineAt(t *testing.T) {
	file := New("first\nsecond\nthird")

	if got := file.LineAt(0); got != 1 {
		t.Fatalf("LineAt(0) = %d, want 1", got)
	}
	if got := file.LineAt(6); got != 2 {
		t.Fatalf("LineAt(6) = %d, want 2", got)
	}
	if got := file.LineAt(999); got != 3 {
		t.Fatalf("LineAt(end) = %d, want 3", got)
	}
}

func TestLineAtClampsNegativeAndLargeOffsets(t *testing.T) {
	file := New("first\nsecond")

	if got := file.LineAt(-1); got != 1 {
		t.Fatalf("LineAt(-1) = %d, want 1", got)
	}
	if got := file.LineAt(999); got != 2 {
		t.Fatalf("LineAt(999) = %d, want 2", got)
	}
}

func TestNewNormalizesReversedLineEnding(t *testing.T) {
	file := New("first\n\rsecond")
	if file.Text != "first\nsecond" {
		t.Fatalf("normalized text = %q, want %q", file.Text, "first\nsecond")
	}
}
