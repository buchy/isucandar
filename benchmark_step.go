package isucandar

import (
	"context"
	"github.com/rosylilly/isucandar/failure"
	"github.com/rosylilly/isucandar/score"
	"sync"
)

type BenchmarkStep struct {
	errorCode failure.Code
	mu        sync.RWMutex
	result    *BenchmarkResult
	cancel    context.CancelFunc
}

func (b *BenchmarkStep) setErrorCode(code failure.Code) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.errorCode = code
}

func (b *BenchmarkStep) AddError(err error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	b.result.Errors.Add(failure.NewError(b.errorCode, err))
}

func (b *BenchmarkStep) AddScore(tag score.ScoreTag) {
	b.result.Score.Add(tag)
}

func (b *BenchmarkStep) Cancel() {
	b.cancel()
}

func (b *BenchmarkStep) wait() {
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		b.result.Score.Wait()
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		b.result.Errors.Wait()
		wg.Done()
	}()
	wg.Wait()
}