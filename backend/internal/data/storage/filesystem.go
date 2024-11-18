package storage

import (
	"context"
	"io"
	"io/fs"

	"github.com/samber/lo"
	"github.com/spf13/afero"
)

type filesystem struct {
	fs afero.Fs
}

func NewFilesystem(source afero.Fs, directory string) Storage {
	return &filesystem{fs: afero.NewBasePathFs(source, directory)}
}

func (f *filesystem) Get(_ context.Context, name string) (io.ReadCloser, error) {
	exist, err := afero.Exists(f.fs, name)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, ErrFileNotFound
	}

	file, err := f.fs.Open(name)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (f *filesystem) List(_ context.Context) ([]io.ReadCloser, error) {
	fileInfos, err := afero.ReadDir(f.fs, "/")
	if err != nil {
		return nil, err
	}

	return lo.FilterMap(fileInfos, func(item fs.FileInfo, _ int) (io.ReadCloser, bool) {
		if item.IsDir() {
			return nil, false
		}
		file, err := f.fs.Open(item.Name())
		if err != nil {
			return nil, false
		}
		return file, true
	}), nil
}

func (f *filesystem) Store(_ context.Context, name string, file io.Reader) error {
	buf, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	if err := afero.WriteFile(f.fs, name, buf, fs.ModePerm); err != nil {
		return err
	}
	return nil
}
