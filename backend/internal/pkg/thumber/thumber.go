package thumber

import (
	"bytes"
	"io"

	"github.com/prplecake/go-thumbnail"
)

//go:generate go run github.com/vektra/mockery/v2
type Thumber interface {
	Thumb(io.Reader) (io.Reader, error)
}

type thumber struct {
	gen thumbnail.Generator
}

func NewThumber() Thumber {
	return &thumber{gen: thumbnail.Generator{
		Scaler: "CatmullRom",
		Width:  64,
		Height: 64,
	}}
}
func (t *thumber) Thumb(in io.Reader) (io.Reader, error) {
	buf, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	img, err := t.gen.NewImageFromByteArray(buf)
	if err != nil {
		return nil, err
	}

	thumBuf, err := t.gen.CreateThumbnail(img)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(thumBuf), nil
}
