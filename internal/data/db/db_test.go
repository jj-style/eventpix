package db_test

import (
	"encoding/base64"
	"testing"

	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func TestCanCreateEventWithSameSlugAfterDeletion_Fix_35(t *testing.T) {
	d, _, err := db.NewDb(&config.Database{
		Driver:        "sqlite",
		Uri:           "file::memory:?cache=shared",
		EncryptionKey: base64.StdEncoding.EncodeToString([]byte("supersecretkeysupersecretkey1234")),
	}, zap.NewNop(), &oauth2.Config{})
	require.NoError(t, err)

	id1, err := d.CreateEvent(t.Context(), &db.Event{Name: "event 1", Slug: "duplicate"})
	require.NoError(t, err)

	require.NoError(t, d.DeleteEvent(t.Context(), uint64(id1)))

	id2, err := d.CreateEvent(t.Context(), &db.Event{Name: "event 2", Slug: "duplicate"})
	require.NoError(t, err)
	require.Equal(t, id1+1, id2)
}
