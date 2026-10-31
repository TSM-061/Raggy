package upload

import "testing"

func TestProgressValidation(t *testing.T) {
	for _, tc := range []struct {
		total, processed int
		valid            bool
	}{{0, 0, true}, {3, 1, true}, {3, 3, true}, {-1, 0, false}, {3, -1, false}, {3, 4, false}} {
		err := (ProgressMessage{Type: Progress, ChunkTotal: tc.total, ChunksProcessed: tc.processed}).Validate()
		if (err == nil) != tc.valid {
			t.Fatalf("%+v: %v", tc, err)
		}
	}
}
