package service

import (
	"errors"
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/jj-style/eventpix/internal/pkg/utils/auth"
)

type AuthService struct {
	db          db.DB
	createToken func(username string) (string, error)
	htmx        *htmx.HTMX
}

func NewAuthService(cfg *config.Config, db db.DB, htmx *htmx.HTMX) *AuthService {
	createToken := func(username string) (string, error) { return auth.CreateToken(cfg.Server.SecretKey, username) }
	return &AuthService{db, createToken, htmx}
}

func (x *AuthService) Login(c *gin.Context) {
	h := x.htmx.NewHandler(c.Writer, c.Request)
	type loginRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var req loginRequest

	if err := c.BindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	user, err := x.db.GetUser(c, req.Username)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	if !auth.ComparePassword(req.Password, user.Password) {
		c.AbortWithError(http.StatusBadRequest, errors.New("invalid password"))
		return
	}

	token, err := x.createToken(user.Username)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	c.SetCookie(auth.CookieName, token, 3600, "/", "", false, true)

	if h.IsHxRequest() {
		h.Redirect("/events")
	}
}

func (x *AuthService) Register(c *gin.Context) {
	h := x.htmx.NewHandler(c.Writer, c.Request)
	type registerRequest struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	var req registerRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if err := x.db.CreateUser(c, req.Username, req.Password); err != nil {
		c.AbortWithError(http.StatusBadRequest, err)
		return
	}

	if h.IsHxRequest() {
		h.Redirect("/login")
	}
}

func (x *AuthService) Logout(c *gin.Context) {
	h := x.htmx.NewHandler(c.Writer, c.Request)
	c.SetCookie(auth.CookieName, "", -1, "/", "localhost", false, true)
	if h.IsHxRequest() {
		h.Redirect("/login")
	} else {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}
}
