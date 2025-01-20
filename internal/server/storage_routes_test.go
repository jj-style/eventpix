package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	service "github.com/jj-style/eventpix/internal/service/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestStorageRoutes(t *testing.T) {
	t.Parallel()
	is := require.New(t)
	router := gin.Default()
	msvc := service.NewMockStorageService(t)
	handleStorage(router.Group("/"), msvc)

	t.Run("happy picture", func(t *testing.T) {
		t.Parallel()

		msvc.EXPECT().
			GetPicture(mock.Anything, "happyPicture").
			Return("file.jpg", []byte("data"), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/picture/happyPicture", nil)
		router.ServeHTTP(w, req)

		is.Equal(200, w.Code)
		is.Equal("data", w.Body.String())
		is.Equal(w.Header().Get("Content-Disposition"), "attachment; filename=file.jpg")
	})

	t.Run("unhappy picture", func(t *testing.T) {
		t.Parallel()

		msvc.EXPECT().
			GetPicture(mock.Anything, "unhappyPicture").
			Return("", []byte(nil), errors.New("boom"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/picture/unhappyPicture", nil)
		router.ServeHTTP(w, req)

		is.Equal(500, w.Code)
	})

	t.Run("happy thumbnail", func(t *testing.T) {
		t.Parallel()

		msvc.EXPECT().
			GetThumbnail(mock.Anything, "happyThumb").
			Return("file.jpg", []byte("data"), nil)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/thumbnail/happyThumb", nil)
		router.ServeHTTP(w, req)

		is.Equal(200, w.Code)
		is.Equal("data", w.Body.String())
		is.Equal(w.Header().Get("Content-Disposition"), "attachment; filename=file.jpg")
	})

	t.Run("unhappy thumbnail", func(t *testing.T) {
		t.Parallel()

		msvc.EXPECT().
			GetThumbnail(mock.Anything, "unhappyThumb").
			Return("", []byte(nil), errors.New("boom"))

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/thumbnail/unhappyThumb", nil)
		router.ServeHTTP(w, req)

		is.Equal(500, w.Code)
	})
}
