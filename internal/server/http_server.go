package server

import (
	"embed"
	"html/template"
	"net/http"
	"time"

	"github.com/donseba/go-htmx"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/pprof"
	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	"github.com/jj-style/eventpix/internal/pkg/validate"
	"github.com/jj-style/eventpix/internal/server/middleware"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

//go:embed assets/static/**/*
var staticFs embed.FS

func NewHttpServer(
	cfg *config.Config,
	htmx *htmx.HTMX,
	storageService service.StorageService,
	authService *service.AuthService,
	eventpixSvc service.EventpixService,
	db db.DB,
	nc *nats.Conn,
	logger *zap.Logger,
	googleOauthConfig *oauth2.Config,
	validator validate.Validator,
) *http.Server {

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{".jpg", ".jpeg", ".png"})))
	pprof.Register(r)

	errorTmpl := template.Must(template.ParseFS(content, "assets/templates/errorToast.html"))
	htmxMiddleware := middleware.Htmx(htmx, errorTmpl)

	authRequired := middleware.AuthRequired(cfg.Server.SecretKey, db)

	authGroup := r.Group("/auth")
	authGroup.Use(htmxMiddleware)
	authGroup.POST("/login", authService.Login)
	if !cfg.Server.SingleEventMode {
		authGroup.POST("/register", authService.Register)
	}
	authGroup.GET("/logout", authRequired, authService.Logout)

	// serve static assets (html/css/js/images)
	staticFsEmbed, err := static.EmbedFolder(staticFs, "assets/static")
	if err != nil {
		logger.Sugar().Fatal(err)
	}
	r.StaticFS("/static", staticFsEmbed)

	// htmx ui / api
	handleUi(r, htmx, db, eventpixSvc, nc, cfg, validator)

	storageGroup := r.Group("/storage")
	handleStorage(storageGroup, storageService)

	oauthGroup := r.Group("/oauth2")
	oauthGroup.Use(authRequired)
	handleOauth(oauthGroup, googleOauthConfig, db, htmxMiddleware)

	// /upload
	uploadGroup := r.Group("/upload")
	uploadGroup.Use(htmxMiddleware)
	{
		setupUploadRoutes(uploadGroup, logger, htmx, eventpixSvc)
	}

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           r,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}
	return server
}
