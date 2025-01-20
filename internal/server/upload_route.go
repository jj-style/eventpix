package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/service"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func setupUploadRoutes(r *gin.RouterGroup, logger *zap.Logger, htmx *htmx.HTMX, svc service.EventpixService) {
	r.POST("", handleUpload(logger.Sugar(), htmx, svc))
}

func handleUpload(log *zap.SugaredLogger, htmx *htmx.HTMX, svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := htmx.NewHandler(c.Writer, c.Request)

		form, err := c.MultipartForm()
		if err != nil {
			AbortWithError(c, http.StatusBadRequest, err)
			return
		}

		if len(form.Value["eventId"]) == 0 {
			AbortWithError(c, http.StatusUnprocessableEntity, errors.New("missing eventId"))
			return
		}
		eventId, err := strconv.ParseUint(form.Value["eventId"][0], 10, 64)
		if err != nil {
			AbortWithError(c, http.StatusBadRequest, fmt.Errorf("parsing eventId: %v", err))
			return
		}
		var g errgroup.Group
		for _, file := range form.File["files"] {
			file := file
			log.Infof("saving file %s", file.Filename)
			g.Go(func() error {
				f, err := file.Open()
				if err != nil {
					return err
				}
				defer f.Close()
				return svc.Upload(c, eventId, file.Filename, f, file.Header.Get("Content-Type"))
			})
		}
		if err := g.Wait(); err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		if h.IsHxRequest() {
			h.TriggerAfterSettle("uploadComplete")
		}
	}
}
