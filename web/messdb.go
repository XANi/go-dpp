package web

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (w *WebBackend) addMessDBAPI() {
	r := w.r.Group("/api/messdb")
	r.GET("/k/:key", w.messdbGet)
	r.POST("/k/:key", w.messdbSet)
}

type JSONError struct {
	Error string `json:"error"`
}

type MessDBKey struct {
	Data string `json:"data"`
}

func newJSONError(err error, str ...string) JSONError {
	if len(str) == 0 {
		return JSONError{Error: "error: " + err.Error()}
	} else {
		return JSONError{Error: strings.Join(str, " ") + ": " + err.Error()}
	}
}

func (w *WebBackend) messdbGet(c *gin.Context) {
	k := c.Param("key")
	data, found, err := w.db.Get(k)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newJSONError(err, "error loading key"))
		return
	}
	if !found {
		c.JSON(http.StatusNotFound, gin.H{})
		return
	}
	c.JSON(http.StatusOK, &MessDBKey{Data: data})
}

func (w *WebBackend) messdbSet(c *gin.Context) {
	k := c.Param("key")
	d := MessDBKey{}
	c.BindJSON(&d)

	err := w.db.Set(k, d.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, newJSONError(err, "error loading key"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
