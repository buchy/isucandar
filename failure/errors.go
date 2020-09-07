package failure

import (
	"context"
	"sync"
)

type Errors struct {
	wg     sync.WaitGroup
	mu     sync.RWMutex
	closer sync.Once
	errors []error
	queue  chan error
}

func NewErrors(ctx context.Context) *Errors {
	set := &Errors{
		wg:     sync.WaitGroup{},
		mu:     sync.RWMutex{},
		closer: sync.Once{},
		errors: make([]error, 0, 0),
		queue:  make(chan error),
	}

	go set.collect(ctx)

	return set
}

func (s *Errors) collect(ctx context.Context) {
	s.wg.Add(1)
	defer s.wg.Done()

	go func() {
		<-ctx.Done()
		s.closer.Do(func() { close(s.queue) })
	}()

	for err := range s.queue {
		s.mu.Lock()
		s.errors = append(s.errors, err)
		s.mu.Unlock()
	}
}

func (s *Errors) Add(err error) {
	defer func() { recover() }()
	s.queue <- err
}

func (s *Errors) Wait() {
	s.wg.Wait()
}

func (s *Errors) Done() {
	s.closer.Do(func() { close(s.queue) })
	s.Wait()
}

func (s *Errors) Messages() map[string][]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(map[string][]string)
	for _, err := range s.errors {
		code := GetErrorCode(err)
		if _, ok := table[code]; ok {
			table[code] = append(table[code], err.Error())
		} else {
			table[code] = []string{err.Error()}
		}
	}

	return table
}

func (s *Errors) Count() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	table := make(map[string]int64)
	for _, err := range s.errors {
		code := GetErrorCode(err)
		if _, ok := table[code]; ok {
			table[code]++
		} else {
			table[code] = 1
		}
	}

	return table
}