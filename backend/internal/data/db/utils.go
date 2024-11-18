package db

import (
	"errors"

	"github.com/jj-style/eventpix/backend/internal/data/storage"
	"github.com/spf13/afero"
)

var ErrNoStorage = errors.New("no storage set on event")

// Sets the events Storage interface value to the non-nil
// storage configuration stored against the event
func ExtractEventStorage(evt *Event) error {
	if st := evt.FileSystemStorage; st != nil {
		evt.Storage = storage.NewFilesystem(afero.NewOsFs(), st.Directory)
	} else {
		return ErrNoStorage
	}
	return nil
}
