package xsoar

import (
	"errors"
	"iter"
	"slices"
)

// ErrEmptyIterator is returned by First when the iterator yields no items.
var ErrEmptyIterator = errors.New("iterator is empty")

// Collect gathers all items from an iterator into a slice.
// It stops on the first error and returns all items collected so far along with the error.
func Collect[T any](seq iter.Seq2[T, error]) ([]T, error) {
	result := make([]T, 0)
	for item, err := range seq {
		if err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, nil
}

// CollectN gathers up to n items from an iterator.
// It stops on the first error and returns all items collected so far along with the error.
func CollectN[T any](seq iter.Seq2[T, error], n int) ([]T, error) {
	result := make([]T, 0, n)
	for item, err := range seq {
		if err != nil {
			return result, err
		}
		result = append(result, item)
		if len(result) >= n {
			break
		}
	}
	return result, nil
}

// First returns the first item from an iterator, or an error if the iterator is empty or fails.
func First[T any](seq iter.Seq2[T, error]) (T, error) {
	for item, err := range seq {
		return item, err
	}
	var zero T
	return zero, ErrEmptyIterator
}

// Take returns an iterator that yields at most n items from the source iterator.
func Take[T any](seq iter.Seq2[T, error], n int) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		count := 0
		for item, err := range seq {
			if !yield(item, err) || err != nil {
				return
			}
			count++
			if count >= n {
				return
			}
		}
	}
}

// Filter returns an iterator that yields only items matching the predicate.
func Filter[T any](seq iter.Seq2[T, error], pred func(T) bool) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for item, err := range seq {
			if err != nil {
				yield(item, err)
				return
			}
			if pred(item) {
				if !yield(item, nil) {
					return
				}
			}
		}
	}
}

// Map transforms each item in the iterator using the provided function.
func Map[T, U any](seq iter.Seq2[T, error], fn func(T) U) iter.Seq2[U, error] {
	return func(yield func(U, error) bool) {
		for item, err := range seq {
			if err != nil {
				var zero U
				yield(zero, err)
				return
			}
			if !yield(fn(item), nil) {
				return
			}
		}
	}
}

// ToSlice converts an iter.Seq to a slice using stdlib slices.Collect.
// This is a convenience wrapper for non-error iterators.
func ToSlice[T any](seq iter.Seq[T]) []T {
	return slices.Collect(seq)
}
