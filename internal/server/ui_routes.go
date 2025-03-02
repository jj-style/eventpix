package server

import (
	"bytes"
	"context"
	"embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/YamiOdymel/multitemplate"
	"github.com/donseba/go-htmx"
	"github.com/g4s8/hexcolor"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	picturev1 "github.com/jj-style/eventpix/internal/gen/picture/v1"
	"github.com/jj-style/eventpix/internal/pkg/validate"
	"github.com/jj-style/eventpix/internal/server/middleware"
	"github.com/jj-style/eventpix/internal/server/sse"
	templatedata "github.com/jj-style/eventpix/internal/server/template_data"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
	"github.com/skip2/go-qrcode"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

//go:embed assets/templates/*
var content embed.FS

func createRenderer() multitemplate.Renderer {
	r := multitemplate.NewRenderer()
	fm := template.FuncMap{
		"isLast": func(index, len int) bool {
			return index+1 == len
		},
		"upper": strings.ToUpper,
	}
	base := "assets/templates/base.html"

	r.AddFromFSFuncs("index", fm, content, base, "assets/templates/index.html")
	r.AddFromFS("eventGallery", content, base, "assets/templates/eventGallery.html")
	r.AddFromFSFuncs("thumbnails", fm, content, "assets/templates/thumbnails.html")

	r.AddFromFS("listEvents", content, base, "assets/templates/eventRow.html", "assets/templates/events.html")
	r.AddFromFS("eventRow", content, "assets/templates/eventRow.html")
	r.AddFromFS("createEvent", content, base, "assets/templates/partials/createEventSlug.html", "assets/templates/createEventForm.html")
	r.AddFromFS("filesystem", content, "assets/templates/forms/filesystem.html")
	r.AddFromFS("s3", content, "assets/templates/forms/s3.html")
	r.AddFromFS("google", content, "assets/templates/forms/google.html")

	r.AddFromFS("login", content, base, "assets/templates/login.html")
	r.AddFromFS("register", content, base, "assets/templates/register.html")
	r.AddFromFS("profile", content, base, "assets/templates/profile.html")

	r.AddFromFS("qrModal", content, "assets/templates/components/qrModal.html")
	r.AddFromFS("createEventSlug", content, "assets/templates/partials/createEventSlug.html")
	return r
}

func handleUi(r *gin.Engine, htmx *htmx.HTMX, db db.DB, svc service.EventpixService, nc *nats.Conn, cfg *config.Config, validator validate.Validator) {
	r.HTMLRender = createRenderer()

	errorTmpl := template.Must(template.ParseFS(content, "assets/templates/errorToast.html"))
	htmxMiddleware := middleware.Htmx(htmx, errorTmpl)

	broker := sse.NewBroker()
	go broker.Listen()

	authRequired := middleware.AuthRequired(cfg.Server.SecretKey, db)
	userEventMiddleware := middleware.UserAuthorizedForEvent(db, "id", "eventId")
	authRedirectMiddleware := middleware.AuthRedirect(cfg.Server.SecretKey, db, "/events")

	// public static pages
	r.GET("/", getIndex())

	// htmx middleware to handle errors nicely
	hr := r.Group("/")
	hr.Use(htmxMiddleware)

	// htmx middleware + authorization required
	hra := r.Group("/")
	hra.Use(htmxMiddleware, authRequired)

	hra.GET("/event/new", getCreateEvent())
	hra.POST("/event", createEvent(svc, htmx))
	hra.GET("/events", getEvents(svc))
	hra.GET("/event/:id/qr/modal", userEventMiddleware, getEventQrModal(svc))
	hra.GET("/event/:id/qr", userEventMiddleware, getQrCode(cfg))
	hra.GET("/profile", getProfile(cfg.OauthSecrets))
	hra.GET("/storageForm", getStorageForm())

	hra.DELETE("/event/:id", userEventMiddleware, deleteEvent(svc))
	hra.POST("/event/:id/live", userEventMiddleware, setEventLive(svc))

	// public view
	hr.GET("/login", authRedirectMiddleware, getLoginForm())
	hr.GET("/register", authRedirectMiddleware, getRegisterForm())
	hr.GET("/event/:id", getEvent(svc))
	hr.GET("/thumbnails/:id", getThumbnails(svc))
	hr.POST("/contact", postContactForm(&http.Client{}, cfg.Server.FormbeeKey))
	hr.POST("/validate/createEvent/slug", postValidateCreateEventSlug(validator))

	// SSE handler and goroutine to listen for new thumbnails and send new data
	hr.GET("/sse", broker.ServeHTTP)
	thumbnailTmpl := template.Must(template.ParseFS(content, "assets/templates/thumbnail.html"))
	go func() {
		sub, err := nc.Subscribe("new-thumbnail", func(msg *nats.Msg) {
			ti, err := svc.GetThumbnailInfo(context.Background(), string(msg.Data))
			if err != nil {
				log.Printf("error getting thumbnail info: %v", err)
				msg.Nak()
				return
			}
			var buf bytes.Buffer
			if err := thumbnailTmpl.Execute(&buf, ti); err != nil {
				log.Printf("error templating sse thumbnail: %v", err)
				msg.Nak()
				return
			}
			broker.Notifier <- sse.NotificationEvent{
				EventName: fmt.Sprintf("new-thumbnail:%d", ti.GetEventId()),
				Payload:   buf.String(),
			}
			msg.Ack()
		})
		if err != nil {
			panic(fmt.Errorf("subscribing thumbnailer sse handler: %v", err))
		}
		defer sub.Drain()
		select {}
	}()
}

// === PARTIALS / HANDLERS ===

func deleteEvent(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId := c.MustGet("eventId").(uint64)
		if _, err := svc.DeleteEvent(c, &picturev1.DeleteEventRequest{Id: eventId}); err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		c.Status(http.StatusOK)
	}
}

func createEvent(svc service.EventpixService, htmx *htmx.HTMX) gin.HandlerFunc {
	bind := func(c *gin.Context, m proto.Message) error {
		if c.Request == nil || c.Request.Body == nil {
			return errors.New("invalid request")
		}
		b, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return errors.New("reading body")
		}
		return protojson.Unmarshal(b, m)
	}

	return func(c *gin.Context) {
		h := htmx.NewHandler(c.Writer, c.Request)
		var req = new(picturev1.CreateEventRequest)
		if err := bind(c, req); err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		user := c.MustGet(gin.AuthUserKey).(*db.User)

		// validation
		switch req.GetStorage().(type) {
		case *picturev1.CreateEventRequest_GoogleDrive:
			if user.GoogleDriveToken == nil {
				AbortWithError(c, http.StatusUnprocessableEntity, errors.New("google drive integration not setup for user"))
				return
			}
		}

		resp, err := svc.CreateEvent(c, user.ID, req)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		if h.IsHxRequest() {
			c.Status(http.StatusCreated)
			h.Redirect("/events")
		} else {
			c.JSON(http.StatusCreated, resp)
		}
	}
}

func setEventLive(svc service.EventpixService) gin.HandlerFunc {
	type tReq struct {
		Live string `json:"live"`
	}
	return func(c *gin.Context) {
		var req tReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		eventId := c.MustGet("eventId").(uint64)
		live := lo.Must(strconv.ParseBool(req.Live))
		evt, err := svc.SetEventLive(c, &picturev1.SetEventLiveRequest{Id: eventId, Live: live})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.HTML(200, "eventRow", evt.GetEvent())
	}
}

func getThumbnails(svc service.EventpixService) gin.HandlerFunc {
	limit := int64(20)
	return func(c *gin.Context) {
		h := c.MustGet(middleware.HtmxKey).(*htmx.Handler)
		pEventId := c.Param("id")
		eventId, err := strconv.ParseUint(pEventId, 10, 64)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		qpage := c.DefaultQuery("page", "0")
		page, err := strconv.ParseInt(qpage, 10, 64)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}

		thumbnails, err := svc.GetThumbnails(c, &picturev1.GetThumbnailsRequest{EventId: eventId, Limit: limit, Offset: limit * page})
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		if h.IsHxRequest() {
			h.TriggerAfterSwap("refreshGallery")
		}

		c.HTML(200, "thumbnails", gin.H{
			"thumbnails": thumbnails.GetThumbnails(),
			"nextPage":   page + 1,
			"eventId":    eventId,
		})
	}
}

func getStorageForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, c.Query("storage"), nil)
	}
}

func postContactForm(client *http.Client, apiKey string) gin.HandlerFunc {
	type request struct {
		Name        string `form:"user" json:"name" binding:"required"`
		Email       string `form:"email" json:"email" binding:"required"`
		PhoneNumber string `form:"phone" json:"phone" binding:"required"`
		Message     string `form:"message" json:"message" binding:"required"`
	}
	url, err := url.JoinPath("https://api.formbee.dev/formbee", apiKey)
	if err != nil {
		panic(err)
	}

	return func(c *gin.Context) {
		var req request
		if err := c.ShouldBindJSON(&req); err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		// any additional validation

		// send back to json for formbee
		body, err := json.Marshal(req)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		ctx, cancel := context.WithTimeout(c, time.Second*5)
		defer cancel()
		formbeeReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}
		formbeeReq.Header.Add("Content-Type", "application/json")
		formbeeReq = formbeeReq.WithContext(ctx)

		if _, err := client.Do(formbeeReq); err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		c.String(http.StatusOK, `<div class="text-center">Thank you for your message. We will be in touch soon.</div>`)
	}
}

func getEventQrModal(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		eventId := c.MustGet("eventId").(uint64)
		event, err := svc.GetEvent(c, &picturev1.GetEventRequest{Value: &picturev1.GetEventRequest_Id{Id: eventId}})
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		c.HTML(http.StatusOK, "qrModal", gin.H{
			"event": event.GetEvent(),
		})
	}
}

func getQrCode(cfg *config.Config) gin.HandlerFunc {
	type request struct {
		Size       int    `form:"size"`
		Foreground string `form:"foreground"`
		Background string `form:"background"`
	}

	return func(c *gin.Context) {
		var req request
		if err := c.Bind(&req); err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		eventId := c.MustGet("eventId").(uint64)
		eventUrl := fmt.Sprintf("%s/event/%d", cfg.Server.ServerUrl, eventId)

		q, err := qrcode.New(eventUrl, qrcode.Medium)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		foreground, err := hexcolor.Parse(req.Foreground)
		if err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}
		background, err := hexcolor.Parse(req.Background)
		if err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}
		q.ForegroundColor = foreground
		q.BackgroundColor = background

		png, err := q.PNG(req.Size)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		b64 := base64.StdEncoding.EncodeToString(png)

		c.String(http.StatusOK, `<img id="eventQrCode" class="mx-auto" src="data:image/png;base64, %s" alt="QR code for %s" />`, b64, eventUrl)

	}
}

// === PAGES ===

func getIndex() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(http.StatusOK, "index", gin.H{
			"title":    "eventpix",
			"features": templatedata.IndexFeatures,
			"pricing":  templatedata.IndexPriceTiers,
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name":   "Home",
						"active": true,
						"href":   "/",
					},
					{
						"name":   "About",
						"active": false,
						"href":   "#features",
					},
					{
						"name":   "Contact",
						"active": false,
						"href":   "#contactForm",
					},
				},
			},
		})
	}
}

func getProfile(oauthCfg *config.OauthSecrets) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(*db.User)
		var googleToken oauth2.Token
		if user.GoogleDriveToken != nil {
			err := json.Unmarshal(user.GoogleDriveToken.Token, &googleToken)
			if err != nil {
				AbortWithError(c, http.StatusInternalServerError, err)
				return
			}
		}
		c.HTML(200, "profile", gin.H{
			"title":       "Profile",
			"user":        user,
			"googleToken": googleToken,
			"oauthConfig": oauthCfg,
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name": "Events",
						"href": "/events",
					},
					{
						"name":         "Profile",
						"href":         "/profile",
						"active":       true,
						"userRequired": true,
					},
					{
						"name":         "Logout",
						"href":         "/auth/logout",
						"userRequired": true,
					},
				},
			},
		})
	}
}

func getLoginForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, "login", gin.H{
			"title": "Login",
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name": "Home",
						"href": "/",
					},
				},
			},
		})
	}
}

func getRegisterForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, "register", gin.H{
			"title": "Register",
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name": "Home",
						"href": "/",
					},
				},
			},
		})
	}
}

func getCreateEvent() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(*db.User)
		stTypes := []struct {
			Name     string
			Value    string
			Disabled bool
		}{
			{Name: "Filesystem", Value: "filesystem"},
			{Name: "S3", Value: "s3"},
			{Name: "Google", Value: "google", Disabled: user.GoogleDriveToken == nil},
		}
		c.HTML(200, "createEvent", gin.H{
			"title": "New Event",
			"user":  c.MustGet(gin.AuthUserKey).(*db.User),
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name": "Events",
						"href": "/events",
					},
					{
						"name":         "Profile",
						"href":         "/profile",
						"userRequired": true,
					},
					{
						"name":         "Logout",
						"href":         "/auth/logout",
						"userRequired": true,
					},
				},
			},
			"storageTypes": stTypes,
		})
	}
}

func getEvents(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := c.MustGet(gin.AuthUserKey).(*db.User)
		events, err := svc.GetEvents(c, &picturev1.GetEventsRequest{}, user.ID)
		if err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.HTML(200, "listEvents", gin.H{
			"title":  "Events",
			"events": events.GetEvents(),
			"user":   user,
			"nav": gin.H{
				"dark": true,
				"items": []gin.H{
					{
						"name":   "Events",
						"active": true,
						"href":   "/events",
					},
					{
						"name":         "Profile",
						"href":         "/profile",
						"userRequired": true,
					},
					{
						"name":         "Logout",
						"href":         "/auth/logout",
						"userRequired": true,
					},
				},
			},
		})
	}
}

func getEvent(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		request := &picturev1.GetEventRequest{}
		pEventId := c.Param("id")
		eventId, err := strconv.ParseUint(pEventId, 10, 64)
		if err != nil {
			// if not an integer, assume it's a slug and try get from that
			request.Value = &picturev1.GetEventRequest_Slug{Slug: pEventId}
		} else {
			request.Value = &picturev1.GetEventRequest_Id{Id: eventId}
		}
		event, err := svc.GetEvent(c, request)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.HTML(http.StatusOK, "eventGallery", gin.H{"title": event.Event.Name, "event": event.Event})
	}
}

func postValidateCreateEventSlug(validator validate.Validator) gin.HandlerFunc {
	type request struct {
		Slug string `json:"slug"`
	}
	return func(c *gin.Context) {
		var req request
		if err := c.ShouldBindBodyWithJSON(&req); err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		class := "valid"
		err := validator.ValidateSlug(req.Slug)
		if err != nil {
			class = "error"
		}
		c.HTML(http.StatusOK, "createEventSlug", gin.H{"slug": req.Slug, "error": err, "class": class})
	}
}
