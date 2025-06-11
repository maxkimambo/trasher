package generator

import (
	"crypto/rand"
	"fmt"
	"sync"
)

// Generator defines the interface for data pattern generators.
type Generator interface {
	Generate(buffer []byte) error
	Name() string
}

// RandomGenerator generates cryptographically secure random data.
type RandomGenerator struct{}

// Name returns the name of the generator.
func (g *RandomGenerator) Name() string {
	return "random"
}

// Generate fills the buffer with cryptographically secure random data.
func (g *RandomGenerator) Generate(buffer []byte) error {
	_, err := rand.Read(buffer)
	return err
}

// SequentialGenerator generates sequential byte patterns.
type SequentialGenerator struct {
	counter uint8
	mu      sync.Mutex
}

// Name returns the name of the generator.
func (g *SequentialGenerator) Name() string {
	return "sequential"
}

// Generate fills the buffer with sequential byte patterns (0x00, 0x01, 0x02, ...).
func (g *SequentialGenerator) Generate(buffer []byte) error {
	g.mu.Lock()
	startValue := g.counter
	g.counter = uint8((int(g.counter) + len(buffer)) % 256)
	g.mu.Unlock()

	for i := range buffer {
		buffer[i] = uint8((int(startValue) + i) % 256)
	}
	return nil
}

// ZeroGenerator fills the buffer with zeros.
type ZeroGenerator struct{}

// Name returns the name of the generator.
func (g *ZeroGenerator) Name() string {
	return "zero"
}

// Generate fills the buffer with zeros.
func (g *ZeroGenerator) Generate(buffer []byte) error {
	for i := range buffer {
		buffer[i] = 0
	}
	return nil
}

// MixedGenerator alternates between random data chunks and zero-filled chunks.
type MixedGenerator struct {
	random      *RandomGenerator
	zero        *ZeroGenerator
	chunkSize   int
	isRandom    bool
	currentPos  int
	mu          sync.Mutex
}

// NewMixedGenerator creates a new MixedGenerator with the specified chunk size.
// If chunkSize is 0, it defaults to 1024 bytes.
func NewMixedGenerator(chunkSize int) *MixedGenerator {
	if chunkSize <= 0 {
		chunkSize = 1024
	}
	return &MixedGenerator{
		random:    &RandomGenerator{},
		zero:      &ZeroGenerator{},
		chunkSize: chunkSize,
		isRandom:  true,
	}
}

// Name returns the name of the generator.
func (g *MixedGenerator) Name() string {
	return "mixed"
}

// Generate fills the buffer alternating between random and zero chunks.
func (g *MixedGenerator) Generate(buffer []byte) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	offset := 0
	for offset < len(buffer) {
		remainingInChunk := g.chunkSize - g.currentPos
		remainingInBuffer := len(buffer) - offset
		copySize := min(remainingInChunk, remainingInBuffer)

		chunk := buffer[offset : offset+copySize]

		var err error
		if g.isRandom {
			err = g.random.Generate(chunk)
		} else {
			err = g.zero.Generate(chunk)
		}
		if err != nil {
			return err
		}

		g.currentPos += copySize
		offset += copySize

		// Switch to next chunk type if current chunk is complete
		if g.currentPos >= g.chunkSize {
			g.isRandom = !g.isRandom
			g.currentPos = 0
		}
	}

	return nil
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewGenerator creates a new generator based on the pattern name.
func NewGenerator(pattern string) (Generator, error) {
	switch pattern {
	case "random":
		return &RandomGenerator{}, nil
	case "sequential":
		return &SequentialGenerator{}, nil
	case "zero":
		return &ZeroGenerator{}, nil
	case "mixed":
		return NewMixedGenerator(1024), nil
	default:
		return nil, fmt.Errorf("unknown pattern: %s", pattern)
	}
}

// AvailablePatterns returns a list of available pattern names.
func AvailablePatterns() []string {
	return []string{"random", "sequential", "zero", "mixed"}
}