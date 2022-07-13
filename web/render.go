package web

import (
	"fmt"
	"github.com/op/go-logging"
	"goji.io/pat"
	"gopkg.in/unrolled/render.v1"
	"html/template"
	"net/http"
	"sync"
)

var log = logging.MustGetLogger("main")

type renderer struct {
	templateCache map[string]*template.Template
	Cache         bool
	Params        map[string]string
	render        *render.Render
	sync.RWMutex
}

func newRenderer() (r *renderer, err error) {
	var v renderer
	v.templateCache = make(map[string]*template.Template)
	v.Cache = true
	v.Params = make(map[string]string)
	v.render = render.New()
	return &v, err
}

func (r *renderer) Handle(w http.ResponseWriter, req *http.Request) {
	page := pat.Param(req, "page")
	r.HandlePage(page, w, req)
}
func (r *renderer) HandleRoot(w http.ResponseWriter, req *http.Request) {
	page := `index.html`
	r.HandlePage(page, w, req)
}

func (r *renderer) HandleStatus(w http.ResponseWriter, req *http.Request) {
	r.render.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (r *renderer) HandlePage(page string, w http.ResponseWriter, req *http.Request) {
	t, err := r.getTpl(page)
	if err != nil {
		fmt.Fprintf(w, "Page %s not found, err:[%+v]", page, err)
		return
	}
	t.Execute(w, r.Params)
}

func (r *renderer) getTpl(name string) (t *template.Template, err error) {
	r.RLock()
	t, ok := r.templateCache[name]
	r.RUnlock()

	if !ok {
		t, err = template.ParseFiles(fmt.Sprintf("templates/%s", name))
		if err != nil {
			return t, err
		}
		if r.Cache {
			r.Lock()
			r.templateCache[name] = t
			r.Unlock()
		}
	}
	return t, err
}
