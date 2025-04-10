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
	"gorm.io/gorm"
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
		// https://stackoverflow.com/a/18276968
		"dict": func(values ...interface{}) (map[string]interface{}, error) {
			if len(values)%2 != 0 {
				return nil, errors.New("invalid dict call")
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					return nil, errors.New("dict keys must be strings")
				}
				dict[key] = values[i+1]
			}
			return dict, nil
		},
	}
	base := "assets/templates/base.html"

	r.AddFromFSFuncs("index", fm, content, base, "assets/templates/index.html")
	r.AddFromFSFuncs("noActiveEvent", fm, content, base, "assets/templates/noActiveEvent.html")
	r.AddFromFS("eventGallery", content, base, "assets/templates/eventGallery.html")
	r.AddFromFSFuncs("thumbnails", fm, content, "assets/templates/thumbnails.html")

	r.AddFromFSFuncs("listEvents", fm, content, base, "assets/templates/eventRow.html", "assets/templates/events.html")
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

	// htmx middleware to handle errors nicely
	hr := r.Group("/")
	hr.Use(htmxMiddleware)

	// htmx middleware + authorization required
	hra := r.Group("/")
	hra.Use(htmxMiddleware, authRequired)

	// public static pages
	if !cfg.Server.SingleEventMode {
		r.GET("/", getIndex())
	} else {
		hr.GET("/", getActiveEvent(svc))
		hr.POST("/event/:id/active", setActiveEvent(svc))
	}

	hra.GET("/event/new", getCreateEvent())
	hra.POST("/event", createEvent(svc, htmx))
	hra.GET("/events", getEvents(svc, cfg.Server))
	hra.GET("/event/:id/qr/modal", userEventMiddleware, getEventQrModal(svc))
	hra.GET("/event/:id/qr", userEventMiddleware, getQrCode(cfg, svc))
	hra.GET("/profile", getProfile(cfg.OauthSecrets))
	hra.GET("/storageForm", getStorageForm())
	hra.GET("/googleDrivePicker", getDrivePicker(cfg.OauthSecrets))

	hra.DELETE("/event/:id", userEventMiddleware, deleteEvent(svc))
	hra.POST("/event/:id/live", userEventMiddleware, setEventLive(svc, cfg.Server))

	// public view
	hr.GET("/login", authRedirectMiddleware, getLoginForm())
	if !cfg.Server.SingleEventMode {
		hr.GET("/register", authRedirectMiddleware, getRegisterForm())
	}
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

func setEventLive(svc service.EventpixService, cfg *config.Server) gin.HandlerFunc {
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

		c.HTML(200, "eventRow", gin.H{"event": evt.GetEvent(), "config": cfg})
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

func getQrCode(cfg *config.Config, svc service.EventpixService) gin.HandlerFunc {
	type request struct {
		Size            int    `form:"size"`
		Foreground      string `form:"foreground"`
		Background      string `form:"background"`
		IncludePassword string `form:"includePassword"`
	}

	return func(c *gin.Context) {
		var req request
		if err := c.Bind(&req); err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		eventId := c.MustGet("eventId").(uint64)
		event, err := svc.GetEvent(c, &picturev1.GetEventRequest{Value: &picturev1.GetEventRequest_Id{Id: eventId}})
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		eventUrlString := fmt.Sprintf("%s/event/%d", cfg.Server.ServerUrl, eventId)

		eventUrl, err := url.Parse(eventUrlString)
		if err != nil {
			AbortWithError(c, http.StatusInternalServerError, err)
			return
		}

		// optionally encode basic auth into QR code so guest can scan straight into event
		if req.IncludePassword == "on" && event.GetEvent().GetPassword() != nil {
			eventUrl.User = url.UserPassword("guest", event.GetEvent().GetPassword().GetValue())
		}

		q, err := qrcode.New(eventUrl.String(), qrcode.Medium)
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

func getActiveEvent(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		event, err := svc.GetActiveEvent(c, &picturev1.GetActiveEventRequest{})
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				c.HTML(http.StatusOK, "noActiveEvent", gin.H{})
				return
			} else {
				c.AbortWithError(http.StatusBadRequest, err)
				return
			}
		}

		if pwd := event.GetEvent().GetPassword(); pwd != nil {
			gin.BasicAuth(gin.Accounts{"guest": pwd.GetValue()})(c)
		}
		c.HTML(http.StatusOK, "eventGallery", gin.H{"title": event.Event.Name, "event": event.Event})
	}
}

func setActiveEvent(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.MustGet(middleware.HtmxKey).(*htmx.Handler)
		pEventId := c.Param("id")
		eventId, err := strconv.ParseUint(pEventId, 10, 64)
		if err != nil {
			AbortWithError(c, http.StatusUnprocessableEntity, err)
			return
		}

		_, err = svc.SetActiveEvent(c, &picturev1.SetActiveEventRequest{Id: eventId})
		if err != nil {
			AbortWithError(c, http.StatusBadRequest, err)
			return
		}
		c.Status(http.StatusOK)
		h.Refresh(true)

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
			googleTokenRaw, _ := base64.StdEncoding.DecodeString(user.GoogleDriveToken.Token.Raw.(string))
			err := json.Unmarshal(googleTokenRaw, &googleToken)
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

func getEvents(svc service.EventpixService, cfg *config.Server) gin.HandlerFunc {
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
			"config": cfg,
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

		if pwd := event.GetEvent().GetPassword(); pwd != nil {
			gin.BasicAuth(gin.Accounts{"guest": pwd.GetValue()})(c)
		}

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

func getDrivePicker(cfg *config.OauthSecrets) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.MustGet(middleware.HtmxKey).(*htmx.Handler)
		user := c.MustGet(gin.AuthUserKey).(*db.User)
		var googleToken oauth2.Token
		if user.GoogleDriveToken != nil {
			gdt, err := base64.StdEncoding.DecodeString(user.GoogleDriveToken.Token.Raw.(string))
			if err != nil {
				AbortWithError(c, http.StatusInternalServerError, err)
				return
			}
			if err := json.Unmarshal(gdt, &googleToken); err != nil {
				AbortWithError(c, http.StatusInternalServerError, err)
				return
			}
		}
		h.TriggerAfterSettle("drivePicker")
		// TODO(JJ) - make this a template?
		c.String(http.StatusOK, `
			<drive-picker app-id="%s" oauth-token="%s" max-items=1>
				<drive-picker-docs-view include-folders=true select-folder-enabled=true mime-types="application/vnd.google-apps.folder"/>
			</drive-picker>`,
			cfg.Google.AppId, googleToken.AccessToken)
	}
}
