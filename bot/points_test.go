package main

import "testing"

func TestScoreDelta(t *testing.T) {
	tests := map[string]int{
		"++": 1,
		"--": -1,
		"—":  -1,
	}
	for operation, expected := range tests {
		actual, err := scoreDelta(operation)
		if err != nil {
			t.Fatalf("scoreDelta(%q) returned an error: %v", operation, err)
		}
		if actual != expected {
			t.Errorf("scoreDelta(%q) = %d, want %d", operation, actual, expected)
		}
	}
	if _, err := scoreDelta("="); err == nil {
		t.Fatal("expected unsupported operation to fail")
	}
}

func TestPointPropertyValue(t *testing.T) {
	for _, value := range []interface{}{int(5), int32(5), int64(5), float32(5), float64(5)} {
		points, err := pointPropertyValue(value)
		if err != nil {
			t.Fatalf("pointPropertyValue(%T) returned an error: %v", value, err)
		}
		if points != 5 {
			t.Errorf("pointPropertyValue(%T) = %f, want 5", value, points)
		}
	}
	if _, err := pointPropertyValue("5"); err == nil {
		t.Fatal("expected invalid Points property type to fail")
	}
}
