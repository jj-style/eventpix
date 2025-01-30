package middleware

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/jj-style/eventpix/internal/pkg/utils/auth"
)

// Middleware to parse and validate an auth cookie.
// If valid, the user is retrieved and added to the gin request context.
func AuthRequired(secretKey string, db db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before request
		cookie, err := c.Cookie(auth.CookieName)
		if err != nil {
			if errors.Is(err, http.ErrNoCookie) {
				c.Redirect(http.StatusTemporaryRedirect, "/login")
				c.Abort()
				return
			}
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		claims, err := auth.VerifyToken(secretKey, cookie)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		subj, err := claims.GetSubject()
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		user, err := db.GetUser(c, subj)
		if err != nil {
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}
		c.Set(gin.AuthUserKey, user)

		// handle request
		c.Next()

		// after request
	}
}

// Middleware to parse and validate an auth cookie.
// If valid, the the request is redirected to the given page
func AuthRedirect(secretKey string, db db.DB, redirect string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// before request
		cookie, err := c.Cookie(auth.CookieName)
		if err != nil {
			c.Next()
			return
		}
		claims, err := auth.VerifyToken(secretKey, cookie)
		if err != nil {
			c.Next()
			return
		}
		subj, err := claims.GetSubject()
		if err != nil {
			c.Next()
			return
		}
		user, err := db.GetUser(c, subj)
		if err != nil {
			c.Next()
			return
		}
		c.Set(gin.AuthUserKey, user)

		c.Redirect(http.StatusTemporaryRedirect, redirect)
		c.Abort()
	}
}

// Middleware to obtain the user from the context set from `AuthRequired`.
// It extracts the event ID from the path parameter and queries to DB to ensure the user is authorized
// to act on the event.
// Arguments
//
// - `eventIdParam` is the URL path parameter to extract the event ID for.
// - `eventIdKey` is the gin context Key to set to pull out (as this function handles parsing and validating) may as well piggy back off it.
//
// Notes
// Must be used after `AuthRequred`
func UserAuthorizedForEvent(d db.DB, eventIdParam, eventIdKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		pEventId := c.Param(eventIdParam)
		eventId, err := strconv.ParseUint(pEventId, 10, 64)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.Set(eventIdKey, eventId)

		user := c.MustGet(gin.AuthUserKey).(*db.User)

		ok, err := d.UserAuthorizedForEvent(c, user.ID, uint(eventId))
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if !ok {
			c.AbortWithError(http.StatusUnauthorized, errors.New("user not authorized for event"))
			return
		}

		c.Next()
	}
}
