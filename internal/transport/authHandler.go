package transport

import (
	"AstralTest/internal/models/entity"
	"AstralTest/internal/models/response"
	"AstralTest/internal/service"
	"AstralTest/pkg/appError"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(service service.AuthService) *AuthHandler {
	return &AuthHandler{
		service: service,
	}
}

type RegisterRequest struct {
	Token    uuid.UUID `json:"token"`
	Login    string    `json:"login"`
	Password string    `json:"pswd"`
}

func writeJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, err error) {
	var (
		respCode    int
		respStatus  int
		respText    string
		customError appError.AppError
	)

	// all service errors should be appError interface, but check that it is true
	if errors.As(err, &customError) {
		respStatus = customError.HTTPStatus()
		respCode = customError.Code()
		respText = customError.Error()
	} else {
		respCode = 500
		respStatus = 500
		respText = "Unknown error: " + err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(respStatus)
	resp := response.Standard{
		Error: &response.ErrorPayload{
			Code: respCode,
			Text: respText,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func (a *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, appError.MethodNotAllowed())
		return
	}

	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, appError.BadRequest("invalid json"))
		return
	}

	reqUser := &entity.User{
		Login:    req.Login,
		Password: req.Password,
	}

	if err := a.service.Register(r.Context(), reqUser, req.Token); err != nil {
		writeError(w, err)
		return
	}

	resp := response.Standard{
		Response: &response.ResponsePayload{
			"login": req.Login,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, appError.MethodNotAllowed())
		return
	}

	var loginRequest entity.User
	if err := json.NewDecoder(r.Body).Decode(&loginRequest); err != nil {
		writeError(w, appError.BadRequest("invalid json"))
		return
	}

	userToken, err := a.service.Login(r.Context(), &loginRequest)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := response.Standard{
		Response: &response.ResponsePayload{
			"token": userToken.String(),
		},
	}
	writeJSON(w, http.StatusOK, resp)
}

func (a *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeError(w, appError.MethodNotAllowed())
		return
	}

	parts := strings.Split(r.URL.Path, "/")
	if len(parts) != 4 || parts[3] == "" {
		writeError(w, appError.BadRequest("bad request"))
		return
	}

	token, err := uuid.Parse(parts[3])
	if err != nil {
		writeError(w, appError.BadRequest("bad user token"))
		return
	}

	err = a.service.Logout(r.Context(), token)
	if err != nil {
		writeError(w, err)
		return
	}

	resp := response.Standard{
		Response: &response.ResponsePayload{
			token.String(): true,
		},
	}
	writeJSON(w, http.StatusOK, resp)
}
