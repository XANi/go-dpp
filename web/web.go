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
	mux := goji.NewMux()
	mux.Handle(pat.Get("/static/*"), http.StripPrefix("/static", http.FileServer(http.Dir(`public/static`))))
	mux.Handle(pat.Get("/apidoc/*"), http.StripPrefix("/apidoc", http.FileServer(http.Dir(`public/apidoc`))))
	mux.HandleFunc(pat.Get("/"), w.r.HandleRoot)
	return &w, err
}

func (w *Web) Listen() error {
	return http.ListenAndServe(w.cfg.ListenAddr, w.mux)
}
