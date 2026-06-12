package index

import (
	"math"
	"testing"
)

func TestVectorRoundTrip(t *testing.T) {
	gotBlob, err := encodeVector([]float32{1, 2.5, -3})
	if err != nil {
		t.Fatalf("encodeVector: %v", err)
	}
	got, err := decodeVector(gotBlob, 3)
	if err != nil {
		t.Fatalf("decodeVector: %v", err)
	}
	want := []float32{1, 2.5, -3}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestDecodeVectorRejectsWrongLength(t *testing.T) {
	if _, err := decodeVector([]byte{1, 2, 3}, 1); err == nil {
		t.Fatal("decodeVector succeeded with wrong blob length")
	}
}

func TestEncodeVectorRejectsNaNAndInfinity(t *testing.T) {
	if _, err := encodeVector([]float32{float32(math.NaN())}); err == nil {
		t.Fatal("encodeVector succeeded with NaN")
	}
	if _, err := encodeVector([]float32{float32(math.Inf(1))}); err == nil {
		t.Fatal("encodeVector succeeded with infinity")
	}
}

func TestCosineSimilarity(t *testing.T) {
	same, err := cosineSimilarity([]float32{1, 2, 3}, []float32{1, 2, 3})
	if err != nil {
		t.Fatalf("cosineSimilarity same: %v", err)
	}
	if math.Abs(same-1) > 1e-6 {
		t.Fatalf("same vector similarity = %v, want 1", same)
	}

	orthogonal, err := cosineSimilarity([]float32{1, 0}, []float32{0, 1})
	if err != nil {
		t.Fatalf("cosineSimilarity orthogonal: %v", err)
	}
	if math.Abs(orthogonal) > 1e-6 {
		t.Fatalf("orthogonal similarity = %v, want 0", orthogonal)
	}
}

func TestCosineSimilarityRejectsInvalidInput(t *testing.T) {
	if _, err := cosineSimilarity(nil, []float32{1}); err == nil {
		t.Fatal("cosineSimilarity succeeded with empty vector")
	}
	if _, err := cosineSimilarity([]float32{0, 0}, []float32{1, 0}); err == nil {
		t.Fatal("cosineSimilarity succeeded with zero-norm vector")
	}
	if _, err := cosineSimilarity([]float32{1}, []float32{1, 2}); err == nil {
		t.Fatal("cosineSimilarity succeeded with dimension mismatch")
	}
}
