package items

import (
	"github.com/go-chi/jwtauth/v5"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type CreateHandler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type ListHandler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type GetHandler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type UpdateHandler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type DeleteHandler struct {
	itemsRepo *repository.ItemRepository
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
	s3client  *s3.Client
}

func NewCreateHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *CreateHandler {
	return &CreateHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewListHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *ListHandler {
	return &ListHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewGetHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *GetHandler {
	return &GetHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewUpdateHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *UpdateHandler {
	return &UpdateHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewDeleteHandler(items *repository.ItemRepository, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth, s3client *s3.Client) *DeleteHandler {
	return &DeleteHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth, s3client: s3client}
}
