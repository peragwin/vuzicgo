package audio

import (
	"context"
	"testing"
	"time"
)

func TestNewSource(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	out, errc := NewSource(ctx, &Config{
		BlockSize: 256, Channels: 1, SampleRate: 44100,
	})
	n := 0

	go func() {
		for {
			select {
			case in := <-out:
				if in == nil {
					t.Fatal("Source terminated early")
				}
			case err := <-errc:
				t.Fatal(err)
			}
			n++
		}
	}()

	<-ctx.Done()

	if n < 10 {
		t.Fatal("Expected at least 10 reads from source")
	}
}
