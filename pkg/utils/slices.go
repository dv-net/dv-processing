package utils //nolint:nolintlint,revive

import "sync"

type Slice[T any] struct {
	mu    *sync.RWMutex
	items []T
}

func NewSlice[T any]() *Slice[T] {
	return &Slice[T]{
		mu:    &sync.RWMutex{},
		items: make([]T, 0),
	}
}

func NewSliceWithLength[T any](length int) *Slice[T] {
	return &Slice[T]{
		mu:    &sync.RWMutex{},
		items: make([]T, length),
	}
}

func (s *Slice[T]) Add(item T) {
	s.mu.Lock()
	s.items = append(s.items, item)
	s.mu.Unlock()
}

func (s *Slice[T]) AddToIndex(index int, item T) {
	s.mu.Lock()
	s.items[index] = item
	s.mu.Unlock()
}

func (s *Slice[T]) AddMany(items []T) {
	s.mu.Lock()
	s.items = append(s.items, items...)
	s.mu.Unlock()
}

func (s *Slice[T]) GetAll() []T {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.items
}

func (s *Slice[T]) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.items)
}
