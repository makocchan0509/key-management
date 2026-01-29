package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

// NewRouter はルーターを生成する。
func NewRouter(h *KeyHandler) http.Handler {
	r := chi.NewRouter()

	// ミドルウェア
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)

	// ルート定義
	r.Route("/v1/tenants/{tenant_id}/keys", func(r chi.Router) {
		r.Post("/", h.CreateKey)
		r.Get("/", h.ListKeys)
		r.Get("/current", h.GetCurrentKey)
		r.Get("/{generation}", h.GetKeyByGeneration)
		r.Delete("/{generation}", h.DisableKey)
		r.Post("/rotate", h.RotateKey)
	})

	return r
}
