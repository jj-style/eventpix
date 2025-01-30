package storage

import (
	"context"
	"fmt"
	"io"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type googleDriveStore struct {
	drive    *drive.Service
	folderId string
}

func (g *googleDriveStore) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	got, err := g.drive.Files.List().Q(fmt.Sprintf("name = '%s'", name)).Do()
	if err != nil {
		return nil, err
	}
	if len(got.Files) == 0 {
		return nil, ErrFileNotFound
	}
	resp, err := g.drive.Files.Get(got.Files[0].Id).Download()
	return resp.Body, err
}

func (g *googleDriveStore) Store(ctx context.Context, name string, data io.Reader) error {
	_, err := g.drive.Files.Create(&drive.File{
		Name:    name,
		Parents: []string{g.folderId},
	}).Media(data).Do()
	return err
}

func NewGoogleDriveStorage(config *oauth2.Config, token *oauth2.Token, folderId string) (Storage, error) {
	client := config.Client(context.Background(), token)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &googleDriveStore{srv, folderId}, nil
}
