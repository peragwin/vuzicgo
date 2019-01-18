package main

import "testing"

func TestTransform(t *testing.T) {
	frameIndex := 0
	rows := 16
	frameSize := 16

	indexInFrames := func(i int) int {
		if i < rows-frameIndex {
			return i + frameIndex
		}
		return i - rows + frameIndex
	}
	rotateIndex := func(i int) int {
		if i < frameSize/2 {
			return i + frameSize/2
		}
		return i - frameSize/2
	}

	for i := 0; i < rows; i++ {
		frameIndex = i
		t.Log("index=", frameIndex)
		for j := 0; j < frameSize; j++ {
			t.Log("j, indexInFrames, rotateIndex =", j, indexInFrames(j), rotateIndex(j))
		}
	}
}
