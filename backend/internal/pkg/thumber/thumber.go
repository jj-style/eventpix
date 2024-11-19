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
}

func NewThumber() Thumber {
	return &thumber{}
}
func (t *thumber) Thumb(in io.Reader) (io.Reader, error) {
	gen := thumbnail.Generator{
		Scaler: "CatmullRom",
		Width:  64,
		Height: 64,
	}
	buf, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}
	img, err := gen.NewImageFromByteArray(buf)
	if err != nil {
		return nil, err
	}

	thumBuf, err := gen.CreateThumbnail(img)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(thumBuf), nil
}
