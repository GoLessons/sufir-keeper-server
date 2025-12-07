package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/GoLessons/sufir-keeper-server/internal/app/httputil"
	"github.com/GoLessons/sufir-keeper-server/internal/auth"
	"github.com/GoLessons/sufir-keeper-server/internal/model"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type RegisterHandler struct{ users *repository.UserRepository }

func NewRegisterHandler(users *repository.UserRepository) *RegisterHandler {
	return &RegisterHandler{users: users}
}

type userRegisterInput struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	var body userRegisterInput
	if err := httputil.DecodeJSON(r, &body); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON")
		return
	}
	login := strings.TrimSpace(body.Login)
	if login == "" || len(login) < 3 || len(body.Password) < 6 {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid login or password")
		return
	}
	_, getErr := h.users.GetByLogin(r.Context(), login)
	if getErr == nil {
		httputil.WriteError(w, http.StatusConflict, "conflict", "User already exists")
		return
	}
	if getErr != sql.ErrNoRows {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	hashed, err := auth.HashPassword(body.Password)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Password hashing error")
		return
	}
	user := model.NewUser(uuid.New(), login, hashed, time.Now().UTC())
	if _, err = h.users.Save(r.Context(), user); err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "server_error", "Database error")
		return
	}
	httputil.WriteJSON(w, http.StatusCreated, map[string]string{"message": "User registered successfully"})
}
