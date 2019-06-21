package util

import "testing"

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(10)
	rb.Push([]float64{1, 2, 3, 4, 5, 6})
	rb.Push([]float64{7, 8, 9, 10, 11, 12})

	g := rb.Get(10)
	exp := []float64{3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	for i := range g {
		if g[i] != exp[i] {
			t.Fatal(exp, g)
		}
	}

	g = rb.GetOffset(10, 2)
	exp = []float64{11, 12, 3, 4, 5, 6, 7, 8, 9, 10}
	for i := range g {
		if g[i] != exp[i] {
			t.Fatal(exp, g)
		}
	}

	g = rb.GetOffset(10, -2)
	exp = []float64{5, 6, 7, 8, 9, 10, 11, 12, 3, 4}
	for i := range g {
		if g[i] != exp[i] {
			t.Fatal(exp, g)
		}
	}
}
