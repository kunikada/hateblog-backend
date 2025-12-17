package tag

import (
	"context"
	"errors"
	"testing"
	"time"

	domainTag "hateblog/internal/domain/tag"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	tag domainTag.Tag
	err error
}

func (f *fakeRepo) GetByName(ctx context.Context, name string) (*domainTag.Tag, error) {
	if f.err != nil {
		return nil, f.err
	}
	result := f.tag
	result.Name = name
	return &result, nil
}

func (f *fakeRepo) List(ctx context.Context, limit, offset int) ([]domainTag.Tag, error) {
	return []domainTag.Tag{f.tag}, nil
}

func (f *fakeRepo) IncrementViewHistory(ctx context.Context, tagID domainTag.ID, viewedAt time.Time) error {
	if tagID == uuid.Nil {
		return errors.New("missing id")
	}
	return nil
}

func TestGetByNameNormalizes(t *testing.T) {
	repo := &fakeRepo{tag: domainTag.Tag{ID: uuid.New(), Name: "go"}}
	svc := NewService(repo, nil)

	result, err := svc.GetByName(context.Background(), "  GoLang ")
	require.NoError(t, err)
	require.Equal(t, "golang", result.Name)

	err = svc.RecordView(context.Background(), repo.tag.ID, time.Now())
	require.NoError(t, err)
}
