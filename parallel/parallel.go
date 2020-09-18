package parallel

import (
	"context"
	"errors"
	"sync/atomic"
	"time"
)

var (
	ErrLimiterClosed = errors.New("limiter closed")
	ErrNegativeCount = errors.New("negative count")
)

const (
	closedFalse uint32 = iota
	closedTrue
)

type Parallel struct {
	ctx    context.Context
	limit  int32
	count  int32
	closed uint32
}

func NewParallel(ctx context.Context, limit int32) *Parallel {
	p := &Parallel{
		ctx:    ctx,
		limit:  limit,
		count:  0,
		closed: closedFalse,
	}

	go func() {
		<-ctx.Done()
		atomic.StoreUint32(&p.closed, closedTrue)
	}()

	return p
}

func (l *Parallel) CurrentLimit() int32 {
	return atomic.LoadInt32(&l.limit)
}

func (l *Parallel) Do(f func(context.Context)) error {
	atomic.AddInt32(&l.count, 1)

	err := l.start()
	if err != nil {
		atomic.AddInt32(&l.count, -1)
		return err
	}

	go func() {
		defer l.done()
		f(l.ctx)
	}()

	return nil
}

func (l *Parallel) Wait() {
	countp := &l.count

	for {
		select {
		case <-l.ctx.Done():
			return
		case <-time.After(time.Microsecond):
			count := atomic.LoadInt32(countp)
			if count == 0 {
				return
			}
		}
	}
}

func (l *Parallel) Close() {
	atomic.StoreUint32(&l.closed, closedTrue)
}

func (l *Parallel) SetParallelism(limit int32) {
	atomic.StoreInt32(&l.limit, limit)
}

func (l *Parallel) AddParallelism(limit int32) {
	atomic.AddInt32(&l.limit, limit)
}

func (l *Parallel) start() error {
	for l.isRunning() {
		if count, kept := l.isLimitKept(); kept {
			if atomic.CompareAndSwapInt32(&l.count, count, count+1) {
				return nil
			}
		}
	}

	return ErrLimiterClosed
}

func (l *Parallel) done() {
	if atomic.AddInt32(&l.count, -2) < 0 {
		panic(ErrNegativeCount)
	}
}

func (l *Parallel) isRunning() bool {
	return atomic.LoadUint32(&l.closed) == closedFalse && l.ctx.Err() == nil
}

func (l *Parallel) isLimitKept() (int32, bool) {
	limit := atomic.LoadInt32(&l.limit)
	count := atomic.LoadInt32(&l.count)
	return count, limit < 1 || count < (limit*2)
}
