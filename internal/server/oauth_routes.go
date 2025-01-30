package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/jj-style/eventpix/internal/server/middleware"
	"golang.org/x/oauth2"
)

func handleOauth(r *gin.RouterGroup, cfg *oauth2.Config, db db.DB, htmxMiddleware gin.HandlerFunc) {
	r.GET("/redirect/google", handleGoogleRedirect(cfg, db))
	r.DELETE("/google", htmxMiddleware, deleteGoogleToken(db))
}

func deleteGoogleToken(db2 db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(*db.User)
		if user.GoogleDriveToken == nil {
			c.Status(http.StatusNoContent)
			return
		}

		if err := db2.DeleteGoogleToken(c, user.ID); err != nil {
			AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("deleting google token for user: %v", err))
			return
		}

		h, _ := c.MustGet(middleware.HtmxKey).(*htmx.Handler)
		if h.IsHxRequest() {
			h.Refresh(true)
		}
		c.Status(http.StatusOK)
	}
}

func handleGoogleRedirect(cfg *oauth2.Config, db2 db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(*db.User)

		// get code from google auth redirect
		code := c.Query("code")
		if code == "" {
			AbortWithError(c, http.StatusBadRequest, errors.New("missing code"))
			return
		}

		// perform token exchange to get access+refresh token
		tok, err := cfg.Exchange(c, code, oauth2.AccessTypeOffline)
		if err != nil {
			AbortWithError(c, 500, fmt.Errorf("unable to retrieve access token: %v", err))
			return
		}

		tokBytes, err := json.Marshal(tok)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("serializing oauth token for storage: %v", err))
			return
		}

		// store token for later use
		if err := db2.StoreGoogleToken(c, user.ID, tokBytes); err != nil {
			AbortWithError(c, http.StatusInternalServerError, fmt.Errorf("storing users access token: %v", err))
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "/profile")
	}
}
