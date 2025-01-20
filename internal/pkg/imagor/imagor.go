package imagor

import (
	"fmt"
	"io"
	"net/http"

	"github.com/jj-style/eventpix/internal/config"
)

// Imagor is a client to the imagorthumbail service
type Imagor interface {
	ThumbImage(imgUrl string) (io.ReadCloser, error)
	ThumbVideo(videoUrl string) (io.ReadCloser, error)
}

type imagor struct {
	url string
}

func NewImagor(cfg *config.Config) Imagor {
	return &imagor{url: cfg.Imagor.Url}
}

func (i *imagor) ThumbImage(imgUrl string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/unsafe/fit-in/200x200/%s", i.url, imgUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		return nil, fmt.Errorf("error creating thumbanil: %d: %s", resp.StatusCode, string(b))
	}
	return resp.Body, nil
}

func (i *imagor) ThumbVideo(videoUrl string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/unsafe/300x0/7x7/filters:label(video,10,10,15,white,20):fill/%s", i.url, videoUrl)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}
