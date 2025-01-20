package server

import (
	"bytes"
	"context"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/YamiOdymel/multitemplate"
	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
	"github.com/jj-style/eventpix/internal/config"
	"github.com/jj-style/eventpix/internal/data/db"
	picturev1 "github.com/jj-style/eventpix/internal/gen/picture/v1"
	"github.com/jj-style/eventpix/internal/server/middleware"
	"github.com/jj-style/eventpix/internal/server/sse"
	"github.com/jj-style/eventpix/internal/service"
	"github.com/nats-io/nats.go"
	"github.com/samber/lo"
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
	}
	r.AddFromFS("eventGallery", content, "assets/templates/base.html", "assets/templates/eventGallery.html")
	r.AddFromFSFuncs("thumbnails", fm, content, "assets/templates/thumbnails.html")

	r.AddFromFS("listEvents", content, "assets/templates/base.html", "assets/templates/eventRow.html", "assets/templates/events.html")
	r.AddFromFS("eventRow", content, "assets/templates/eventRow.html")
	r.AddFromFS("createEvent", content, "assets/templates/base.html", "assets/templates/createEventForm.html")
	r.AddFromFS("filesystem", content, "assets/templates/forms/filesystem.html")
	r.AddFromFS("s3", content, "assets/templates/forms/s3.html")
	r.AddFromFS("login", content, "assets/templates/base.html", "assets/templates/login.html")
	r.AddFromFS("register", content, "assets/templates/base.html", "assets/templates/register.html")
	return r
}

func handleUi(r *gin.Engine, htmx *htmx.HTMX, db db.DB, svc service.EventpixService, nc *nats.Conn, cfg *config.Config) {
	r.HTMLRender = createRenderer()

	errorTmpl := template.Must(template.ParseFS(content, "assets/templates/errorToast.html"))
	htmxMiddleware := middleware.Htmx(htmx, errorTmpl)

	broker := sse.NewBroker()
	go broker.Listen()

	authRequired := middleware.AuthRequired(cfg.Server.SecretKey, db)
	userEventMiddleware := middleware.UserAuthorizedForEvent(db, "id", "eventId")

	// htmx middleware to handle errors nicely
	hr := r.Group("/")
	hr.Use(htmxMiddleware)

	// htmx middleware + authorization required
	hra := r.Group("/")
	hra.Use(htmxMiddleware, authRequired)

	hra.GET("/event/new", getCreateEvent())
	hra.POST("/event", createEvent(svc, htmx))
	hra.GET("/events", getEvents(svc))
	hra.GET("/storageForm", getStorageForm())

	hra.DELETE("/event/:id", userEventMiddleware, deleteEvent(svc))
	hra.POST("/event/:id/live", userEventMiddleware, setEventLive(svc))

	// public view
	hr.GET("/login", getLoginForm())
	hr.GET("/register", getRegisterForm())
	hr.GET("/event/:id", getEvent(svc))
	hr.GET("/thumbnails/:id", getThumbnails(svc))

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

func getEvent(svc service.EventpixService) gin.HandlerFunc {
	return func(c *gin.Context) {
		pEventId := c.Param("id")
		eventId, err := strconv.ParseUint(pEventId, 10, 64)
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		event, err := svc.GetEvent(c, &picturev1.GetEventRequest{Id: eventId})
		if err != nil {
			c.AbortWithError(http.StatusBadRequest, err)
			return
		}
		c.HTML(http.StatusOK, "eventGallery", gin.H{"title": event.Event.Name, "event": event.Event})
	}
}

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
			"user":   lo.Must(c.Get(gin.AuthUserKey)),
		})
	}
}

func getCreateEvent() gin.HandlerFunc {
	stTypes := []struct {
		Name  string
		Value string
	}{
		{Name: "Filesystem", Value: "filesystem"},
		{Name: "S3", Value: "s3"},
	}
	return func(c *gin.Context) {
		c.HTML(200, "createEvent", gin.H{"title": "New Event", "storageTypes": stTypes})
	}
}

func getStorageForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, c.Query("storage"), nil)
	}
}

func getLoginForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, "login", gin.H{"title": "Login"})
	}
}

func getRegisterForm() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.HTML(200, "register", gin.H{"title": "Register"})
	}
}
