package storage

import (
	"context"
	"io"

	"golang.org/x/oauth2"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

type googleDriveStore struct {
	drive    *drive.Service
	folderId string
}

func (g *googleDriveStore) Get(ctx context.Context, id string) (io.ReadCloser, error) {
	resp, err := g.drive.Files.Get(id).Download()
	if err != nil {
		return nil, err
	}
	return resp.Body, err
}

func (g *googleDriveStore) Store(ctx context.Context, name string, data io.Reader) (string, error) {
	f, err := g.drive.Files.Create(&drive.File{
		Name:    name,
		Parents: []string{g.folderId},
	}).Media(data).Do()
	if err != nil {
		return "", err
	}
	return f.Id, err
}

func NewGoogleDriveStorage(config *oauth2.Config, token *oauth2.Token, folderId string) (Storage, error) {
	client := config.Client(context.Background(), token)
	srv, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	return &googleDriveStore{srv, folderId}, nil
}
