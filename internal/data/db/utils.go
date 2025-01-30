package db

import (
	"encoding/json"
	"errors"

	"github.com/jj-style/eventpix/internal/data/storage"
	"github.com/spf13/afero"
	"golang.org/x/oauth2"
)

var ErrNoStorage = errors.New("no storage set on event")

// Sets the events Storage interface value to the non-nil
// storage configuration stored against the event
func ExtractEventStorage(evt *Event, googleOauthConfig *oauth2.Config) error {
	if st := evt.FileSystemStorage; st != nil {
		evt.Storage = storage.NewFilesystem(afero.NewOsFs(), st.Directory)
	} else if st := evt.S3Storage; st != nil {
		evt.Storage = storage.NewS3Store(&storage.S3Config{
			Region:    st.Region,
			AccessKey: st.AccessKey,
			SecretKey: st.SecretKey,
			Bucket:    st.Bucket,
			Endpoint:  st.Endpoint,
		})
	} else if st := evt.GoogleDriveStorage; st != nil {
		var token oauth2.Token
		json.Unmarshal(evt.User.GoogleDriveToken.Token, &token)
		gdrive, err := storage.NewGoogleDriveStorage(googleOauthConfig, &token, st.DirectoryID)
		if err != nil {
			return err
		}
		evt.Storage = gdrive
	} else {
		return ErrNoStorage
	}
	return nil
}
