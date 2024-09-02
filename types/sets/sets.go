package sets

import (
	"slices"

	"golang.org/x/exp/constraints"
)

type OrderedComparable interface {
	constraints.Ordered
	comparable
}

type Set[T OrderedComparable] struct {
	values map[T]bool
}

func New[T constraints.Ordered](ts ...T) *Set[T] {
	values := make(map[T]bool)
	for _, t := range ts {
		values[t] = true
	}

	return &Set[T]{
		values: values,
	}
}

func (s *Set[T]) Add(ts ...T) {
	for _, t := range ts {
		s.values[t] = true
	}
}

func (s *Set[T]) Delete(t T) {
	delete(s.values, t)
}

func (s *Set[T]) Len() int {
	return len(s.values)
}

func (s *Set[T]) Has(t T) bool {
	return s.values[t]
}

func (s *Set[T]) All() []T {
	res := make([]T, 0, len(s.values))
	for t := range s.values {
		res = append(res, t)
	}

	slices.Sort(res)

	return res
}

func (s *Set[T]) Intersect(other *Set[T]) *Set[T] {
	if s.Len() > other.Len() {
		return other.Intersect(s)
	}

	set := New[T]()
	for t := range s.values {
		if other.Has(t) {
			set.Add(t)
		}
	}

	return set
}

func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	set := New(append(s.All(), other.All()...)...)

	return set
}

func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	set := New(s.All()...)

	for _, t := range other.All() {
		set.Delete(t)
	}

	return set
}

func (s *Set[T]) Equal(other *Set[T]) bool {
	for t := range s.values {
		if !other.Has(t) {
			return false
		}
	}

	return s.Len() == other.Len()
}
