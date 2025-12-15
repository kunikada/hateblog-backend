package redis

import (
	"testing"

	"hateblog/internal/domain/entry"

	"github.com/stretchr/testify/require"
)

func TestBuildKey(t *testing.T) {
	cache := &EntryListCache{}
	key1, err := cache.BuildKey(entry.ListQuery{
		Sort:             entry.SortHot,
		MinBookmarkCount: 10,
		Limit:            50,
		Offset:           5,
		Tags:             []string{"Go", "Web"},
	})
	require.NoError(t, err)

	key2, err := cache.BuildKey(entry.ListQuery{
		Sort:             entry.SortHot,
		MinBookmarkCount: 10,
		Limit:            50,
		Offset:           5,
		Tags:             []string{" web ", "go"},
	})
	require.NoError(t, err)

	require.Equal(t, key1, key2)
}
