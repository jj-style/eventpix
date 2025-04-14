package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
)

type ftpStore struct {
	cfg *FtpConfig
}

func (f *ftpStore) Get(ctx context.Context, name string) (io.ReadCloser, error) {
	conn, err := f.login()
	if err != nil {
		return nil, err
	}
	defer conn.Logout()
	file, err := conn.Retr(name)
	if err != nil {
		if strings.HasPrefix(err.Error(), fmt.Sprint(ftp.StatusFileUnavailable)) {
			return nil, ErrFileNotFound
		}
		return nil, err
	}
	return file, nil
}

func (f *ftpStore) Store(ctx context.Context, name string, file io.Reader) (string, error) {
	conn, err := f.login()
	if err != nil {
		return "", err
	}
	defer conn.Logout()
	if err := conn.Stor(name, file); err != nil {
		return "", err
	}
	return name, nil
}

func (f *ftpStore) login() (*ftp.ServerConn, error) {
	c, err := ftp.Dial(f.cfg.Address, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return nil, err
	}
	if err := c.Login(f.cfg.Username, f.cfg.Password); err != nil {
		return nil, err
	}

	if err := c.ChangeDir(f.cfg.Directory); err != nil {
		return nil, err
	}
	return c, nil
}

type FtpConfig struct {
	Username  string
	Password  string
	Address   string
	Directory string
}

func NewFtpStore(cfg *FtpConfig) (Storage, error) {

	return &ftpStore{cfg: cfg}, nil
}
