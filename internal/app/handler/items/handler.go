package items

import (
	"github.com/go-chi/jwtauth/v5"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type Handler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

func NewHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *Handler {
	return &Handler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}
