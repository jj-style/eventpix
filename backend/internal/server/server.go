package server

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/grpchealth"
	"connectrpc.com/grpcreflect"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jj-style/eventpix/backend/gen/picture/v1/picturev1connect"
	"github.com/jj-style/eventpix/backend/internal/config"
	"github.com/jj-style/eventpix/backend/internal/service"
	"github.com/rs/cors"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func NewServer(
	cfg *config.Config,
	pictureService picturev1connect.PictureServiceHandler,
	storageService *service.StorageService,
	logger *zap.Logger,
) *http.Server {

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5, "image/jpeg", "image/png"))
	r.Mount("/debug", middleware.Profiler())

	// TODO: custom endpoint to get thumbnail/image image
	r.Route("/storage", func(r chi.Router) {
		r.Get("/thumbnail/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			got, err := storageService.GetThumbnail(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(got)
			w.Header().Add("Cache-Control", "max-age=3600") // proxies to cache for 1 hour
		})
		r.Get("/picture/{id}", func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "id")
			got, err := storageService.GetPicture(r.Context(), id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Write(got)
			w.Header().Add("Cache-Control", "max-age=3600") // proxies to cache for 1 hour
		})
	})

	compress1KB := connect.WithCompressMinBytes(1024)
	path, handler := picturev1connect.NewPictureServiceHandler(pictureService, compress1KB)
	r.Mount(path, handler)
	r.Mount(grpchealth.NewHandler(
		grpchealth.NewStaticChecker(picturev1connect.PictureServiceName),
		compress1KB,
	))
	r.Mount(grpcreflect.NewHandlerV1(
		grpcreflect.NewStaticReflector(picturev1connect.PictureServiceName),
		compress1KB,
	))

	// Use h2c so we can serve HTTP/2 without TLS.
	h := h2c.NewHandler(newCORS().Handler(r), &http2.Server{})

	server := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           h,
		ReadHeaderTimeout: time.Second,
		ReadTimeout:       5 * time.Minute,
		WriteTimeout:      5 * time.Minute,
		MaxHeaderBytes:    8 * 1024, // 8KiB
	}
	return server
}

func newCORS() *cors.Cors {
	// To let web developers play with the demo service from browsers, we need a
	// very permissive CORS setup.
	return cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowOriginFunc: func(_ /* origin */ string) bool {
			// Allow all origins, which effectively disables CORS.
			return true
		},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{
			// Content-Type is in the default safelist.
			"Accept",
			"Accept-Encoding",
			"Accept-Post",
			"Connect-Accept-Encoding",
			"Connect-Content-Encoding",
			"Content-Encoding",
			"Grpc-Accept-Encoding",
			"Grpc-Encoding",
			"Grpc-Message",
			"Grpc-Status",
			"Grpc-Status-Details-Bin",
		},
		// Let browsers cache CORS information for longer, which reduces the number
		// of preflight requests. Any changes to ExposedHeaders won't take effect
		// until the cached data expires. FF caps this value at 24h, and modern
		// Chrome caps it at 2h.
		MaxAge: int(2 * time.Hour / time.Second),
	})
}
