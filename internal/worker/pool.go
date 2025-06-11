package worker

import (
	"context"
	"runtime"
	"sync"

	"github.com/maxkimambo/trasher/pkg/generator"
)

// WorkerPool manages a pool of worker goroutines for parallel data generation.
type WorkerPool struct {
	numWorkers int
	chunkSize  int64
	workChan   chan workItem
	resultChan chan resultItem
	errorChan  chan error
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	bufferPool sync.Pool
}

// workItem represents a unit of work to be processed by a worker.
type workItem struct {
	offset int64
	size   int64
}

// resultItem represents the result of processed work.
type resultItem struct {
	buffer []byte
	offset int64
}

// NewWorkerPool creates a new worker pool with the specified configuration.
// If numWorkers is 0 or negative, it defaults to runtime.NumCPU().
// If chunkSize is 0 or negative, it defaults to 64MB.
func NewWorkerPool(ctx context.Context, numWorkers int, chunkSize int64) *WorkerPool {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if chunkSize <= 0 {
		chunkSize = 64 * 1024 * 1024 // 64MB default
	}

	ctx, cancel := context.WithCancel(ctx)
	
	pool := &WorkerPool{
		numWorkers: numWorkers,
		chunkSize:  chunkSize,
		workChan:   make(chan workItem, numWorkers*2),
		resultChan: make(chan resultItem, numWorkers*2),
		errorChan:  make(chan error, numWorkers),
		ctx:        ctx,
		cancel:     cancel,
	}

	// Initialize buffer pool
	pool.bufferPool = sync.Pool{
		New: func() interface{} {
			buffer := make([]byte, chunkSize)
			return &buffer
		},
	}

	return pool
}

// Start begins the worker pool processing with the given generator and total size.
func (p *WorkerPool) Start(gen generator.Generator, totalSize int64) {
	// Start worker goroutines
	for i := 0; i < p.numWorkers; i++ {
		p.wg.Add(1)
		go p.worker(gen)
	}

	// Start work distributor goroutine
	go p.distributeWork(totalSize)
}

// worker is the main worker goroutine that processes work items.
func (p *WorkerPool) worker(gen generator.Generator) {
	defer p.wg.Done()

	for {
		select {
		case <-p.ctx.Done():
			return
		case work, ok := <-p.workChan:
			if !ok {
				return
			}

			// Get buffer from pool
			bufferPtr := p.bufferPool.Get().(*[]byte)
			buffer := *bufferPtr

			// Resize buffer if needed for last chunk
			if work.size < int64(len(buffer)) {
				buffer = buffer[:work.size]
			}

			// Generate data
			if err := gen.Generate(buffer); err != nil {
				select {
				case p.errorChan <- err:
				case <-p.ctx.Done():
				}
				p.cancel()
				p.bufferPool.Put(bufferPtr)
				return
			}

			// Send result
			select {
			case <-p.ctx.Done():
				p.bufferPool.Put(bufferPtr)
				return
			case p.resultChan <- resultItem{buffer: buffer, offset: work.offset}:
				// Buffer will be returned to pool after processing
			}
		}
	}
}

// distributeWork creates and distributes work items to workers.
func (p *WorkerPool) distributeWork(totalSize int64) {
	defer close(p.workChan)

	var offset int64
	for offset < totalSize {
		size := p.chunkSize
		if remaining := totalSize - offset; remaining < size {
			size = remaining
		}

		select {
		case <-p.ctx.Done():
			return
		case p.workChan <- workItem{offset: offset, size: size}:
			offset += size
		}
	}
}

// Results returns the result channel for reading processed chunks.
func (p *WorkerPool) Results() <-chan resultItem {
	return p.resultChan
}

// Errors returns the error channel for reading worker errors.
func (p *WorkerPool) Errors() <-chan error {
	return p.errorChan
}

// ReturnBuffer returns a buffer to the pool for reuse.
func (p *WorkerPool) ReturnBuffer(buffer []byte) {
	p.bufferPool.Put(&buffer)
}

// Wait waits for all workers to complete and closes result channels.
func (p *WorkerPool) Wait() {
	p.wg.Wait()
	close(p.resultChan)
	close(p.errorChan)
}

// Shutdown gracefully shuts down the worker pool.
func (p *WorkerPool) Shutdown() {
	p.cancel()
	p.Wait()
}

// NumWorkers returns the number of workers in the pool.
func (p *WorkerPool) NumWorkers() int {
	return p.numWorkers
}

// ChunkSize returns the chunk size used by the worker pool.
func (p *WorkerPool) ChunkSize() int64 {
	return p.chunkSize
}