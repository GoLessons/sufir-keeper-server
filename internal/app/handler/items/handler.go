package items

import (
	"github.com/go-chi/jwtauth/v5"

	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type CreateHandler struct {
	itemsRepo repository.ItemStore
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type ListHandler struct {
	itemsRepo repository.ItemStore
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type GetHandler struct {
	itemsRepo repository.ItemStore
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type UpdateHandler struct {
	itemsRepo repository.ItemStore
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
}

type DeleteHandler struct {
	itemsRepo repository.ItemStore
	kek       keyencrypt.Provider
	tokenAuth *jwtauth.JWTAuth
	s3client  s3.Service
}

func NewCreateHandler(items repository.ItemStore, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *CreateHandler {
	return &CreateHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewListHandler(items repository.ItemStore, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *ListHandler {
	return &ListHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewGetHandler(items repository.ItemStore, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *GetHandler {
	return &GetHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewUpdateHandler(items repository.ItemStore, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth) *UpdateHandler {
	return &UpdateHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth}
}

func NewDeleteHandler(items repository.ItemStore, kek keyencrypt.Provider, tokenAuth *jwtauth.JWTAuth, s3client s3.Service) *DeleteHandler {
	return &DeleteHandler{itemsRepo: items, kek: kek, tokenAuth: tokenAuth, s3client: s3client}
}
