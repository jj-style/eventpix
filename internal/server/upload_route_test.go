package server

import (
	"bytes"
	"embed"
	"errors"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	mockService "github.com/jj-style/eventpix/internal/service/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//go:embed test/data
var testData embed.FS

func TestUploadRoute(t *testing.T) {
	t.Parallel()
	is := require.New(t)
	router := gin.Default()
	msvc := mockService.NewMockEventpixService(t)
	setupUploadRoutes(router.Group("/upload"), zap.NewNop(), htmx.New(), msvc)

	t.Run("missing eventId", func(t *testing.T) {
		t.Parallel()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		multipartFilesUpload(t, writer, "files", testData, []string{"test/data/0.jpg", "test/data/1.jpg"})
		err := writer.Close()
		is.NoError(err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		router.ServeHTTP(w, req)

		is.Equal(422, w.Code)
	})

	t.Run("happy", func(t *testing.T) {
		t.Parallel()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("eventId", "1")
		multipartFilesUpload(t, writer, "files", testData, []string{"test/data/0.jpg", "test/data/1.jpg"})
		err := writer.Close()
		is.NoError(err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		// expect 2 uploads to happen
		msvc.EXPECT().Upload(mock.Anything, uint64(1), "0.jpg", mock.Anything, "application/octet-stream").Return(nil)
		msvc.EXPECT().Upload(mock.Anything, uint64(1), "1.jpg", mock.Anything, "application/octet-stream").Return(nil)

		router.ServeHTTP(w, req)

		is.Equal(200, w.Code)
	})

	t.Run("error uploading", func(t *testing.T) {
		t.Parallel()

		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("eventId", "2")
		multipartFilesUpload(t, writer, "files", testData, []string{"test/data/0.jpg", "test/data/1.jpg"})
		err := writer.Close()
		is.NoError(err)

		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/upload", body)
		req.Header.Add("Content-Type", writer.FormDataContentType())

		// expect 2 uploads to happen
		msvc.EXPECT().Upload(mock.Anything, uint64(2), "0.jpg", mock.Anything, "application/octet-stream").Return(nil)
		msvc.EXPECT().Upload(mock.Anything, uint64(2), "1.jpg", mock.Anything, "application/octet-stream").Return(errors.New("boom"))

		router.ServeHTTP(w, req)

		is.Equal(500, w.Code)
	})
}

func multipartFilesUpload(t *testing.T, writer *multipart.Writer, paramName string, fs fs.FS, filenames []string) {
	t.Helper()
	is := require.New(t)

	for _, filePath := range filenames {
		file, err := fs.Open(filePath)
		is.NoError(err)
		defer file.Close()
		part, err := writer.CreateFormFile(paramName, filepath.Base(filePath))
		is.NoError(err)
		_, err = io.Copy(part, file)
		is.NoError(err)
	}
}
