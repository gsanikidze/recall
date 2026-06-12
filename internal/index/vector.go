package index

import (
	"encoding/binary"
	"fmt"
	"math"
)

func encodeVector(v []float32) ([]byte, error) {
	blob := make([]byte, len(v)*4)
	for i, value := range v {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return nil, fmt.Errorf("invalid vector value at index %d", i)
		}
		binary.LittleEndian.PutUint32(blob[i*4:(i+1)*4], math.Float32bits(value))
	}
	return blob, nil
}

func decodeVector(blob []byte, dim int) ([]float32, error) {
	if dim < 0 {
		return nil, fmt.Errorf("invalid vector dimension %d", dim)
	}
	if len(blob) != dim*4 {
		return nil, fmt.Errorf("vector blob length %d does not match dimension %d", len(blob), dim)
	}
	v := make([]float32, dim)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(blob[i*4 : (i+1)*4]))
	}
	return v, nil
}

func cosineSimilarity(a, b []float32) (float64, error) {
	if len(a) == 0 || len(b) == 0 {
		return 0, fmt.Errorf("vectors must not be empty")
	}
	if len(a) != len(b) {
		return 0, fmt.Errorf("vector dimension mismatch: %d != %d", len(a), len(b))
	}

	var dot, normA, normB float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		normA += av * av
		normB += bv * bv
	}
	if normA == 0 || normB == 0 {
		return 0, fmt.Errorf("vectors must have non-zero norm")
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB)), nil
}
