package db

import (
	"encoding/base64"
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
			AccessKey: st.AccessKey.Raw.(string),
			SecretKey: st.SecretKey.Raw.(string),
			Bucket:    st.Bucket,
			Endpoint:  st.Endpoint,
			Insecure:  st.Insecure,
		})
	} else if st := evt.GoogleDriveStorage; st != nil {
		var token oauth2.Token
		googleTokenRaw, _ := base64.StdEncoding.DecodeString(evt.User.GoogleDriveToken.Token.Raw.(string))
		json.Unmarshal(googleTokenRaw, &token)
		gdrive, err := storage.NewGoogleDriveStorage(googleOauthConfig, &token, st.DirectoryID)
		if err != nil {
			return err
		}
		evt.Storage = gdrive
	} else if st := evt.FtpStorage; st != nil {
		ftp, err := storage.NewFtpStore(&storage.FtpConfig{
			Address:   st.Address,
			Directory: st.Directory,
			Username:  st.Username.Raw.(string),
			Password:  st.Password.Raw.(string),
		})
		if err != nil {
			return err
		}
		evt.Storage = ftp
	} else {
		return ErrNoStorage
	}
	return nil
}
