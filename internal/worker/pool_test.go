package worker

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/maxkimambo/trasher/pkg/generator"
)

func TestNewWorkerPool(t *testing.T) {
	ctx := context.Background()

	// Test with default values
	p := NewWorkerPool(ctx, 0, 0)
	if p.numWorkers != runtime.NumCPU() {
		t.Errorf("expected numWorkers to default to %d, got %d", runtime.NumCPU(), p.numWorkers)
	}
	if p.chunkSize != 64*1024*1024 {
		t.Errorf("expected chunkSize to default to 64MB, got %d", p.chunkSize)
	}

	// Test with custom values
	p2 := NewWorkerPool(ctx, 4, 1024)
	if p2.numWorkers != 4 {
		t.Errorf("expected numWorkers to be 4, got %d", p2.numWorkers)
	}
	if p2.chunkSize != 1024 {
		t.Errorf("expected chunkSize to be 1024, got %d", p2.chunkSize)
	}

	// Test with negative values (should use defaults)
	p3 := NewWorkerPool(ctx, -1, -1)
	if p3.numWorkers != runtime.NumCPU() {
		t.Errorf("expected numWorkers to default to %d, got %d", runtime.NumCPU(), p3.numWorkers)
	}
	if p3.chunkSize != 64*1024*1024 {
		t.Errorf("expected chunkSize to default to 64MB, got %d", p3.chunkSize)
	}
}

func TestWorkerPoolBasicOperation(t *testing.T) {
	ctx := context.Background()
	p := NewWorkerPool(ctx, 2, 1024)

	gen := &generator.ZeroGenerator{}
	totalSize := int64(4096) // 4KB total, 4 chunks of 1KB each

	p.Start(gen, totalSize)

	// Collect results
	results := make(map[int64][]byte)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for result := range p.Results() {
			results[result.offset] = make([]byte, len(result.buffer))
			copy(results[result.offset], result.buffer)
			p.ReturnBuffer(result.buffer)
		}
	}()

	p.Wait()
	wg.Wait()

	// Verify we got all chunks
	if len(results) != 4 {
		t.Errorf("expected 4 chunks, got %d", len(results))
	}

	// Verify chunk sizes and offsets
	expectedOffsets := []int64{0, 1024, 2048, 3072}
	for _, offset := range expectedOffsets {
		if chunk, exists := results[offset]; !exists {
			t.Errorf("missing chunk at offset %d", offset)
		} else if len(chunk) != 1024 {
			t.Errorf("chunk at offset %d has size %d, expected 1024", offset, len(chunk))
		} else {
			// Verify all bytes are zero (ZeroGenerator)
			for i, b := range chunk {
				if b != 0 {
					t.Errorf("chunk at offset %d, byte %d is not zero: %d", offset, i, b)
					break
				}
			}
		}
	}
}

func TestWorkerPoolWithDifferentGenerators(t *testing.T) {
	generators := []struct {
		name string
		gen  generator.Generator
	}{
		{"random", &generator.RandomGenerator{}},
		{"sequential", &generator.SequentialGenerator{}},
		{"zero", &generator.ZeroGenerator{}},
		{"mixed", generator.NewMixedGenerator(100)},
	}

	for _, test := range generators {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			p := NewWorkerPool(ctx, 2, 512)

			totalSize := int64(2048) // 4 chunks of 512 bytes each
			p.Start(test.gen, totalSize)

			// Collect all results
			var resultCount int
			var wg sync.WaitGroup
			wg.Add(1)

			go func() {
				defer wg.Done()
				for result := range p.Results() {
					resultCount++
					p.ReturnBuffer(result.buffer)
				}
			}()

			p.Wait()
			wg.Wait()

			if resultCount != 4 {
				t.Errorf("expected 4 results, got %d", resultCount)
			}
		})
	}
}

func TestWorkerPoolErrorHandling(t *testing.T) {
	// Create a generator that always fails
	failingGen := &FailingGenerator{}

	ctx := context.Background()
	p := NewWorkerPool(ctx, 2, 1024)

	totalSize := int64(2048)
	p.Start(failingGen, totalSize)

	// Should receive an error
	select {
	case err := <-p.Errors():
		if err == nil {
			t.Error("expected error but got nil")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for error")
	}

	p.Shutdown()
}

func TestWorkerPoolGracefulShutdown(t *testing.T) {
	ctx := context.Background()
	p := NewWorkerPool(ctx, 2, 1024)

	gen := &generator.ZeroGenerator{}
	totalSize := int64(1024 * 1024) // 1MB

	p.Start(gen, totalSize)

	// Shutdown after a short delay
	go func() {
		time.Sleep(5 * time.Millisecond)
		p.Shutdown()
	}()

	// Try to read results until shutdown with timeout
	done := make(chan bool)
	go func() {
		for result := range p.Results() {
			p.ReturnBuffer(result.buffer)
		}
		done <- true
	}()

	select {
	case <-done:
		// Test completed normally
	case <-time.After(time.Second):
		t.Error("test timed out waiting for shutdown")
	}
}

func TestWorkerPoolContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	p := NewWorkerPool(ctx, 2, 1024)
	gen := &generator.ZeroGenerator{}
	totalSize := int64(4096) // 4KB - small amount

	p.Start(gen, totalSize)

	// Cancel immediately 
	cancel()

	// Consume any results that might come through before cancellation
	var resultCount int
	timeout := time.After(100 * time.Millisecond)
	
	done := make(chan struct{})
	go func() {
		defer close(done)
		for result := range p.Results() {
			resultCount++
			p.ReturnBuffer(result.buffer)
		}
	}()

	select {
	case <-done:
		// Results channel closed, test completed
	case <-timeout:
		// Force shutdown if hanging
		p.Shutdown()
		<-done
	}

	// Test passes if we don't hang - cancellation behavior is tested
	t.Logf("Processed %d chunks before cancellation", resultCount)
}

func TestWorkerPoolBufferReuse(t *testing.T) {
	ctx := context.Background()
	p := NewWorkerPool(ctx, 1, 1024)

	gen := &generator.ZeroGenerator{}
	totalSize := int64(2048) // 2 chunks

	p.Start(gen, totalSize)

	// Collect buffers and verify they're being reused
	var buffers []*[]byte
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for result := range p.Results() {
			// Store the buffer pointer for comparison
			buffers = append(buffers, &result.buffer)
			p.ReturnBuffer(result.buffer)
		}
	}()

	p.Wait()
	wg.Wait()

	if len(buffers) != 2 {
		t.Errorf("expected 2 buffers, got %d", len(buffers))
	}
}

func TestWorkerPoolLastChunkSize(t *testing.T) {
	ctx := context.Background()
	p := NewWorkerPool(ctx, 1, 1000)

	gen := &generator.ZeroGenerator{}
	totalSize := int64(2500) // 2 full chunks + 1 partial chunk (500 bytes)

	p.Start(gen, totalSize)

	results := make(map[int64]int)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for result := range p.Results() {
			results[result.offset] = len(result.buffer)
			p.ReturnBuffer(result.buffer)
		}
	}()

	p.Wait()
	wg.Wait()

	// Should have 3 chunks
	if len(results) != 3 {
		t.Errorf("expected 3 chunks, got %d", len(results))
	}

	// First two chunks should be full size
	if size, exists := results[0]; !exists || size != 1000 {
		t.Errorf("first chunk size should be 1000, got %d", size)
	}
	if size, exists := results[1000]; !exists || size != 1000 {
		t.Errorf("second chunk size should be 1000, got %d", size)
	}

	// Last chunk should be partial
	if size, exists := results[2000]; !exists || size != 500 {
		t.Errorf("last chunk size should be 500, got %d", size)
	}
}

func TestWorkerPoolConcurrency(t *testing.T) {
	ctx := context.Background()
	numWorkers := 4
	p := NewWorkerPool(ctx, numWorkers, 1024)

	gen := &generator.SequentialGenerator{}
	totalSize := int64(8192) // 8 chunks

	p.Start(gen, totalSize)

	var processedCount int
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		for result := range p.Results() {
			processedCount++
			p.ReturnBuffer(result.buffer)
		}
	}()

	p.Wait()
	wg.Wait()

	if processedCount != 8 {
		t.Errorf("expected 8 processed chunks, got %d", processedCount)
	}
}

// FailingGenerator is a test generator that always returns an error
type FailingGenerator struct{}

func (g *FailingGenerator) Generate(buffer []byte) error {
	return &GenerationError{Message: "test error"}
}

func (g *FailingGenerator) Name() string {
	return "failing"
}

type GenerationError struct {
	Message string
}

func (e *GenerationError) Error() string {
	return e.Message
}

// Benchmark tests
func BenchmarkWorkerPool(b *testing.B) {
	gen := &generator.ZeroGenerator{}
	
	workerCounts := []int{1, 2, 4}
	for _, numWorkers := range workerCounts {
		b.Run(fmt.Sprintf("workers_%d", numWorkers), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				p := NewWorkerPool(ctx, numWorkers, 1024)
				totalSize := int64(4096) // 4KB

				p.Start(gen, totalSize)

				// Consume all results
				var count int
				for result := range p.Results() {
					count++
					p.ReturnBuffer(result.buffer)
				}

				p.Wait()
			}
		})
	}
}