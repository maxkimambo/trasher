package generator

import (
	"bytes"
	"fmt"
	"sync"
	"testing"
)

func TestRandomGenerator(t *testing.T) {
	g := &RandomGenerator{}

	// Test name
	if g.Name() != "random" {
		t.Errorf("expected name 'random', got %s", g.Name())
	}

	// Test generation with different buffer sizes
	sizes := []int{1, 16, 256, 1024, 4096}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			buffer := make([]byte, size)
			err := g.Generate(buffer)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Check that buffer is not all zeros (very unlikely for random data)
			allZeros := true
			for _, b := range buffer {
				if b != 0 {
					allZeros = false
					break
				}
			}
			if allZeros && size > 1 {
				t.Error("Generated data appears to be all zeros, which is highly unlikely for random data")
			}
		})
	}

	// Test that consecutive calls produce different results
	buffer1 := make([]byte, 100)
	buffer2 := make([]byte, 100)
	
	g.Generate(buffer1)
	g.Generate(buffer2)
	
	if bytes.Equal(buffer1, buffer2) {
		t.Error("Two consecutive random generations produced identical results")
	}
}

func TestSequentialGenerator(t *testing.T) {
	g := &SequentialGenerator{}

	// Test name
	if g.Name() != "sequential" {
		t.Errorf("expected name 'sequential', got %s", g.Name())
	}

	// Test basic sequential pattern
	buffer := make([]byte, 10)
	err := g.Generate(buffer)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !bytes.Equal(buffer, expected) {
		t.Errorf("expected %v, got %v", expected, buffer)
	}

	// Test that counter continues from previous state
	buffer2 := make([]byte, 5)
	err = g.Generate(buffer2)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected2 := []byte{10, 11, 12, 13, 14}
	if !bytes.Equal(buffer2, expected2) {
		t.Errorf("expected %v, got %v", expected2, buffer2)
	}

	// Test wraparound behavior
	g.counter = 254
	buffer3 := make([]byte, 5)
	err = g.Generate(buffer3)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	expected3 := []byte{254, 255, 0, 1, 2}
	if !bytes.Equal(buffer3, expected3) {
		t.Errorf("expected %v, got %v", expected3, buffer3)
	}
}

func TestZeroGenerator(t *testing.T) {
	g := &ZeroGenerator{}

	// Test name
	if g.Name() != "zero" {
		t.Errorf("expected name 'zero', got %s", g.Name())
	}

	// Test generation with different buffer sizes
	sizes := []int{1, 16, 256, 1024}
	for _, size := range sizes {
		t.Run(fmt.Sprintf("size_%d", size), func(t *testing.T) {
			buffer := make([]byte, size)
			// Fill with non-zero data first
			for i := range buffer {
				buffer[i] = 0xFF
			}

			err := g.Generate(buffer)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			// Check that all bytes are zero
			for i, b := range buffer {
				if b != 0 {
					t.Errorf("byte at position %d is not zero: %d", i, b)
				}
			}
		})
	}
}

func TestMixedGenerator(t *testing.T) {
	g := NewMixedGenerator(10) // Small chunk size for testing

	// Test name
	if g.Name() != "mixed" {
		t.Errorf("expected name 'mixed', got %s", g.Name())
	}

	// Test that it produces alternating patterns
	buffer := make([]byte, 30) // 3 chunks of 10 bytes each
	err := g.Generate(buffer)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// First chunk should be random (not all zeros)
	chunk1 := buffer[0:10]
	allZeros1 := true
	for _, b := range chunk1 {
		if b != 0 {
			allZeros1 = false
			break
		}
	}
	if allZeros1 {
		t.Error("First chunk should be random, but appears to be all zeros")
	}

	// Second chunk should be zeros
	chunk2 := buffer[10:20]
	for i, b := range chunk2 {
		if b != 0 {
			t.Errorf("Second chunk byte at position %d should be zero: %d", i, b)
		}
	}

	// Third chunk should be random again
	chunk3 := buffer[20:30]
	allZeros3 := true
	for _, b := range chunk3 {
		if b != 0 {
			allZeros3 = false
			break
		}
	}
	if allZeros3 {
		t.Error("Third chunk should be random, but appears to be all zeros")
	}
}

func TestMixedGeneratorDefaultChunkSize(t *testing.T) {
	g := NewMixedGenerator(0) // Should default to 1024
	if g.chunkSize != 1024 {
		t.Errorf("expected default chunk size 1024, got %d", g.chunkSize)
	}

	g2 := NewMixedGenerator(-1) // Should default to 1024
	if g2.chunkSize != 1024 {
		t.Errorf("expected default chunk size 1024, got %d", g2.chunkSize)
	}
}

func TestNewGenerator(t *testing.T) {
	tests := []struct {
		pattern   string
		expectErr bool
		typeName  string
	}{
		{"random", false, "random"},
		{"sequential", false, "sequential"},
		{"zero", false, "zero"},
		{"mixed", false, "mixed"},
		{"invalid", true, ""},
	}

	for _, tt := range tests {
		t.Run(tt.pattern, func(t *testing.T) {
			g, err := NewGenerator(tt.pattern)
			
			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if g.Name() != tt.typeName {
				t.Errorf("expected generator name %s, got %s", tt.typeName, g.Name())
			}
		})
	}
}

func TestAvailablePatterns(t *testing.T) {
	patterns := AvailablePatterns()
	expected := []string{"random", "sequential", "zero", "mixed"}

	if len(patterns) != len(expected) {
		t.Errorf("expected %d patterns, got %d", len(expected), len(patterns))
	}

	for i, pattern := range patterns {
		if pattern != expected[i] {
			t.Errorf("expected pattern %s at position %d, got %s", expected[i], i, pattern)
		}
	}
}

// Test thread safety
func TestGeneratorThreadSafety(t *testing.T) {
	generators := []struct {
		name string
		gen  Generator
	}{
		{"random", &RandomGenerator{}},
		{"sequential", &SequentialGenerator{}},
		{"zero", &ZeroGenerator{}},
		{"mixed", NewMixedGenerator(100)},
	}

	for _, test := range generators {
		t.Run(test.name, func(t *testing.T) {
			var wg sync.WaitGroup
			numGoroutines := 10
			bufferSize := 100

			for i := 0; i < numGoroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					buffer := make([]byte, bufferSize)
					err := test.gen.Generate(buffer)
					if err != nil {
						t.Errorf("Generate failed in goroutine: %v", err)
					}
				}()
			}

			wg.Wait()
		})
	}
}

// Benchmark tests
func BenchmarkRandomGenerator(b *testing.B) {
	g := &RandomGenerator{}
	buffer := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate(buffer)
	}
}

func BenchmarkSequentialGenerator(b *testing.B) {
	g := &SequentialGenerator{}
	buffer := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate(buffer)
	}
}

func BenchmarkZeroGenerator(b *testing.B) {
	g := &ZeroGenerator{}
	buffer := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate(buffer)
	}
}

func BenchmarkMixedGenerator(b *testing.B) {
	g := NewMixedGenerator(1024)
	buffer := make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		g.Generate(buffer)
	}
}