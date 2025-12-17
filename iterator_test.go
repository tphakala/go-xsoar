package xsoar_test

import (
	"errors"
	"iter"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tphakala/go-xsoar"
)

func makeSeq[T any](items []T) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for _, item := range items {
			if !yield(item, nil) {
				return
			}
		}
	}
}

func makeSeqWithError[T any](items []T, errAt int, err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for i, item := range items {
			if i == errAt {
				var zero T
				yield(zero, err)
				return
			}
			if !yield(item, nil) {
				return
			}
		}
	}
}

func TestCollect(t *testing.T) {
	t.Run("collects all items", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3, 4, 5})

		result, err := xsoar.Collect(seq)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, result)
	})

	t.Run("stops on error", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]int{1, 2, 3, 4, 5}, 3, testErr)

		result, err := xsoar.Collect(seq)
		require.ErrorIs(t, err, testErr)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("handles empty sequence", func(t *testing.T) {
		seq := makeSeq([]int{})

		result, err := xsoar.Collect(seq)
		require.NoError(t, err)
		assert.Empty(t, result)
	})
}

func TestCollectN(t *testing.T) {
	t.Run("collects up to n items", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3, 4, 5})

		result, err := xsoar.CollectN(seq, 3)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("collects all if less than n", func(t *testing.T) {
		seq := makeSeq([]int{1, 2})

		result, err := xsoar.CollectN(seq, 5)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, result)
	})

	t.Run("stops on error before n", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]int{1, 2, 3, 4, 5}, 2, testErr)

		result, err := xsoar.CollectN(seq, 5)
		require.ErrorIs(t, err, testErr)
		assert.Equal(t, []int{1, 2}, result)
	})
}

func TestFirst(t *testing.T) {
	t.Run("returns first item", func(t *testing.T) {
		seq := makeSeq([]string{"a", "b", "c"})

		result, err := xsoar.First(seq)
		require.NoError(t, err)
		assert.Equal(t, "a", result)
	})

	t.Run("returns error for empty iterator", func(t *testing.T) {
		seq := makeSeq([]string{})

		_, err := xsoar.First(seq)
		require.Error(t, err)
		assert.ErrorIs(t, err, xsoar.ErrEmptyIterator)
	})

	t.Run("returns error if first item errors", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]string{"a"}, 0, testErr)

		_, err := xsoar.First(seq)
		require.ErrorIs(t, err, testErr)
	})
}

func TestTake(t *testing.T) {
	t.Run("takes n items", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3, 4, 5})
		taken := xsoar.Take(seq, 3)

		result, err := xsoar.Collect(taken)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, result)
	})

	t.Run("takes all if less than n", func(t *testing.T) {
		seq := makeSeq([]int{1, 2})
		taken := xsoar.Take(seq, 5)

		result, err := xsoar.Collect(taken)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, result)
	})

	t.Run("propagates errors", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]int{1, 2, 3, 4, 5}, 2, testErr)
		taken := xsoar.Take(seq, 5)

		_, err := xsoar.Collect(taken)
		require.ErrorIs(t, err, testErr)
	})
}

func TestFilter(t *testing.T) {
	t.Run("filters items", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3, 4, 5, 6})
		even := xsoar.Filter(seq, func(n int) bool { return n%2 == 0 })

		result, err := xsoar.Collect(even)
		require.NoError(t, err)
		assert.Equal(t, []int{2, 4, 6}, result)
	})

	t.Run("handles no matches", func(t *testing.T) {
		seq := makeSeq([]int{1, 3, 5})
		even := xsoar.Filter(seq, func(n int) bool { return n%2 == 0 })

		result, err := xsoar.Collect(even)
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("propagates errors", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]int{1, 2, 3}, 1, testErr)
		filtered := xsoar.Filter(seq, func(n int) bool { return true })

		_, err := xsoar.Collect(filtered)
		require.ErrorIs(t, err, testErr)
	})
}

func TestMap(t *testing.T) {
	t.Run("transforms items", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3})
		doubled := xsoar.Map(seq, func(n int) int { return n * 2 })

		result, err := xsoar.Collect(doubled)
		require.NoError(t, err)
		assert.Equal(t, []int{2, 4, 6}, result)
	})

	t.Run("transforms to different type", func(t *testing.T) {
		seq := makeSeq([]int{1, 2, 3})
		strings := xsoar.Map(seq, func(n int) string {
			return string(rune('a' + n - 1))
		})

		result, err := xsoar.Collect(strings)
		require.NoError(t, err)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("propagates errors", func(t *testing.T) {
		testErr := errors.New("test error")
		seq := makeSeqWithError([]int{1, 2, 3}, 1, testErr)
		mapped := xsoar.Map(seq, func(n int) int { return n * 2 })

		_, err := xsoar.Collect(mapped)
		require.ErrorIs(t, err, testErr)
	})
}

func TestIteratorComposition(t *testing.T) {
	// Test that iterators can be composed
	seq := makeSeq([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})

	// Filter even, double them, take first 3
	result, err := xsoar.Collect(
		xsoar.Take(
			xsoar.Map(
				xsoar.Filter(seq, func(n int) bool { return n%2 == 0 }),
				func(n int) int { return n * 2 },
			),
			3,
		),
	)

	require.NoError(t, err)
	assert.Equal(t, []int{4, 8, 12}, result) // 2*2, 4*2, 6*2
}
