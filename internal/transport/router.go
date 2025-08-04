package transport

import (
	"AstralTest/internal/service"
	"AstralTest/pkg/appError"
	"net/http"
)

type Handler struct {
	authService service.AuthService
	wcsService  service.WcsService
}

func NewHandler(authService service.AuthService, wcsService service.WcsService) *Handler {
	return &Handler{
		authService: authService,
		wcsService:  wcsService,
	}
}

// TODO: better use different router
func (h *Handler) InitRouter() *http.ServeMux {
	mux := http.NewServeMux()

	authHandler := NewAuthHandler(h.authService)

	mux.HandleFunc("/api/register", authHandler.Register)
	mux.HandleFunc("/api/auth", authHandler.Login)
	mux.HandleFunc("/api/auth/", authHandler.Logout)

	wcsHandler := NewWcsHandler(h.wcsService)

	mux.HandleFunc("/api/docs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			wcsHandler.UploadFileHandler(w, r)
		} else if r.Method == http.MethodGet || r.Method == http.MethodHead {
			wcsHandler.GetFilesList(w, r)
		} else {
			writeError(w, appError.MethodNotAllowed())
		}
	})
	mux.HandleFunc("/api/docs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		}
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			wcsHandler.GetFileHandler(w, r)
		} else if r.Method == http.MethodDelete {
			wcsHandler.DeleteDoc(w, r)
		} else {
			writeError(w, appError.MethodNotAllowed())
		}
	})

	return mux
}
