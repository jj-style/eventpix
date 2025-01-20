package middleware

import (
	"bytes"
	"html/template"
	"io"
	"net/http"

	"github.com/donseba/go-htmx"
	"github.com/gin-gonic/gin"
)

const HtmxKey = "__htmx_ctx_key__"

func Htmx2(htmx *htmx.HTMX, tmpl *template.Template) gin.HandlerFunc {
	return func(c *gin.Context) {
		// create htmx helper for the request
		h := htmx.NewHandler(c.Writer, c.Request)
		c.Set(HtmxKey, h)

		c.Next()

		// if any api errors and an htmx request, put error into an HTMX response for a swap
		if len(c.Errors) > 0 && h.IsHxRequest() {
			var errMsg string
			for _, err := range c.Errors {
				errMsg += err.Error()
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, map[string]any{
				"title":       "Error",
				"status":      c.Writer.Status(),
				"description": errMsg,
			}); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
			h.ReTarget("#globalErrors")
			h.ReSwap("innerHTML")
			c.Header("TEST", "VALUE")
			c.Writer.Header().Set("TEST2", "VALUE")
			c.Data(c.Writer.Status(), "text/html", buf.Bytes())
		}
	}
}

func Htmx(htmx *htmx.HTMX, tmpl *template.Template) gin.HandlerFunc {
	return func(c *gin.Context) {
		wb := &bodyWriter{
			body:           &bytes.Buffer{},
			ResponseWriter: c.Writer,
		}
		c.Writer = wb

		// create htmx helper for the request
		h := htmx.NewHandler(c.Writer, c.Request)
		c.Set(HtmxKey, h)

		c.Next()

		// if any api errors and an htmx request, put error into an HTMX response for a swap
		if len(c.Errors) > 0 && h.IsHxRequest() {
			var errMsg string
			for _, err := range c.Errors {
				errMsg += err.Error()
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, map[string]any{
				"title":       "Error",
				"status":      c.Writer.Status(),
				"description": errMsg,
			}); err != nil {
				c.AbortWithError(http.StatusInternalServerError, err)
				return
			}
			h.ReSwap("none")
			h.TriggerAfterSettle("showErrorToast")
			io.Copy(wb.ResponseWriter, bytes.NewReader(buf.Bytes()))
		} else {
			io.Copy(wb.ResponseWriter, wb.body)
		}
	}
}

type bodyWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (r bodyWriter) Write(b []byte) (int, error) {
	return r.body.Write(b)
}
