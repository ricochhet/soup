package sync

import (
	"sync"
)

type Safe[T any] struct {
	Mutex sync.Mutex
	t     *T
}

func NewSafe[T any]() *Safe[T]          { return &Safe[T]{} }
func (s *Safe[T]) Get() *T              { return s.t }
func (s *Safe[T]) Set(t *T)             { s.t = t }
func (s *Safe[T]) Copy(target *Safe[T]) { s.SetLocked(target.GetLocked()) }

func (s *Safe[T]) GetLocked() *T {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	return s.t
}

func (s *Safe[T]) SetLocked(t *T) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	s.t = t
}

func Get[K comparable, V any](s *Safe[map[K]V], key K) (V, bool) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	m := s.Get()
	if m == nil {
		var zero V
		return zero, false
	}

	v, ok := (*m)[key]

	return v, ok
}

func Set[K comparable, V any](s *Safe[map[K]V], key K, value V) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	m := s.Get()
	if m == nil {
		s.Set(&map[K]V{key: value})
		return
	}

	(*m)[key] = value
}
