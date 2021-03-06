package web


import (
	"goji.io"
	"goji.io/pat"
	"github.com/XANi/go-dpp/config"
	"net/http"
)


type Web struct {
	cfg *config.Config
	r *renderer
	mux *goji.Mux

}


func New(c *config.Config) (*Web, error) {
	var w Web
	r, err := newRenderer()
	if err != nil {
		return nil, err
	}
	w.r = r
	w.cfg = c
	w.mux = goji.NewMux()
	w.mux.Handle(pat.Get("/static/*"), http.StripPrefix("/static", http.FileServer(http.Dir(`public/static`))))
	w.mux.Handle(pat.Get("/apidoc/*"), http.StripPrefix("/apidoc", http.FileServer(http.Dir(`public/apidoc`))))
	w.mux.HandleFunc(pat.Get("/"), w.r.HandleRoot)
	return &w, err
}

func (w *Web) Listen() {
	go func() {
		log.Errorf("Error starting listener: %s", http.ListenAndServe(w.cfg.ListenAddr, w.mux))
	}()
}
