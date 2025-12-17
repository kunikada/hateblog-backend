package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	domainEntry "hateblog/internal/domain/entry"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeEntryStore struct {
	err error
}

func (f *fakeEntryStore) Get(ctx context.Context, id domainEntry.ID) (*domainEntry.Entry, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &domainEntry.Entry{ID: id, Title: "ok", URL: "https://example.com", PostedAt: time.Now()}, nil
}

type fakeClickRepo struct {
	err error
}

func (f *fakeClickRepo) Increment(ctx context.Context, entryID domainEntry.ID, clickedAt time.Time) error {
	return f.err
}

func TestRecordClickValidatesEntry(t *testing.T) {
	entryRepo := &fakeEntryStore{}
	clickRepo := &fakeClickRepo{}
	svc := NewService(entryRepo, clickRepo)

	err := svc.RecordClick(context.Background(), (domainEntry.ID)(uuid.Nil))
	require.Error(t, err)

	entryRepo.err = errors.New("missing")
	err = svc.RecordClick(context.Background(), domainEntry.ID(uuid.New()))
	require.Error(t, err)
}
