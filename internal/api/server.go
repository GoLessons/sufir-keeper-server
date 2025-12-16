package api

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	apiTypes "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"

	authhandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/auth"
	fileshandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/files"
	itemshandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/items"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
	"github.com/GoLessons/sufir-keeper-server/internal/s3"
)

type ServerDependencies struct {
	DatabaseClient         interface{}
	KEKProvider            interface{}
	Logger                 *zap.Logger
	TokenAuth              *jwtauth.JWTAuth
	UsersRepository        *repository.UserRepository
	ItemsRepository        *repository.ItemRepository
	S3Client               *s3.Client
	AccessTokenTTLSeconds  int
	RefreshTokenTTLSeconds int
}

type Server struct {
	Unimplemented
	kek           keyencrypt.Provider
	users         *repository.UserRepository
	items         *repository.ItemRepository
	login         *authhandler.LoginHandler
	refresh       *authhandler.RefreshHandler
	register      *authhandler.RegisterHandler
	logout        *authhandler.LogoutHandler
	itemsCreate   *itemshandler.CreateHandler
	itemsList     *itemshandler.ListHandler
	itemsGet      *itemshandler.GetHandler
	itemsUpdate   *itemshandler.UpdateHandler
	itemsDelete   *itemshandler.DeleteHandler
	filesDownload *fileshandler.DownloadHandler
	filesPresign  *fileshandler.PresignHandler
	verify        *authhandler.VerifyHandler
}

func NewServer(deps ServerDependencies) *Server {
	users := deps.UsersRepository
	items := deps.ItemsRepository
	var kekProvider keyencrypt.Provider
	if p, ok := deps.KEKProvider.(keyencrypt.Provider); ok {
		kekProvider = p
	}
	srv := &Server{
		users:         users,
		items:         items,
		login:         authhandler.NewLoginHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		refresh:       authhandler.NewRefreshHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		register:      authhandler.NewRegisterHandler(users),
		logout:        authhandler.NewLogoutHandler(users, deps.TokenAuth),
		itemsCreate:   nil,
		itemsList:     itemshandler.NewListHandler(items, kekProvider, deps.TokenAuth),
		itemsGet:      nil,
		itemsUpdate:   nil,
		itemsDelete:   itemshandler.NewDeleteHandler(items, kekProvider, deps.TokenAuth, deps.S3Client),
		filesDownload: nil,
		filesPresign:  nil,
		verify:        authhandler.NewVerifyHandler(),
		kek:           kekProvider,
	}
	if kekProvider != nil {
		srv.itemsCreate = itemshandler.NewCreateHandler(items, kekProvider, deps.TokenAuth)
		srv.itemsGet = itemshandler.NewGetHandler(items, kekProvider, deps.TokenAuth)
		srv.itemsUpdate = itemshandler.NewUpdateHandler(items, kekProvider, deps.TokenAuth)
		srv.filesDownload = fileshandler.NewDownloadHandler(items, deps.S3Client, kekProvider)
	}
	if deps.S3Client != nil {
		srv.filesPresign = fileshandler.NewPresignHandler(deps.S3Client)
	}
	return srv
}

func (s *Server) LogoutUser(w http.ResponseWriter, r *http.Request)     { s.logout.Handle(w, r) }
func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request)   { s.refresh.Handle(w, r) }
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request)      { s.login.Handle(w, r) }
func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request)   { s.register.Handle(w, r) }
func (s *Server) AuthVerifyGet(w http.ResponseWriter, r *http.Request)  { s.verify.Handle(w, r) }
func (s *Server) AuthVerifyPost(w http.ResponseWriter, r *http.Request) { s.verify.Handle(w, r) }

func (s *Server) CreateItem(w http.ResponseWriter, r *http.Request) {
	if s.itemsCreate == nil {
		http.Error(w, "kek not available", http.StatusServiceUnavailable)
		return
	}
	s.itemsCreate.Handle(w, r)
}

func (s *Server) GetItems(w http.ResponseWriter, r *http.Request, params GetItemsParams) {
	s.itemsList.Handle(w, r, params)
}

func (s *Server) GetItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	if s.itemsGet == nil {
		http.Error(w, "kek not available", http.StatusServiceUnavailable)
		return
	}
	s.itemsGet.Handle(w, r, uuid.UUID(id))
}

func (s *Server) UpdateItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	if s.itemsUpdate == nil {
		http.Error(w, "kek not available", http.StatusServiceUnavailable)
		return
	}
	s.itemsUpdate.Handle(w, r, uuid.UUID(id))
}

func (s *Server) DeleteItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	s.itemsDelete.Handle(w, r, uuid.UUID(id))
}

func (s *Server) UploadFile(w http.ResponseWriter, _ *http.Request, _ UploadFileParams) {
	http.Error(w, "not implemented", http.StatusNotImplemented)
}

func (s *Server) PresignFile(w http.ResponseWriter, r *http.Request) {
	if s.filesPresign == nil {
		http.Error(w, "presign not available", http.StatusServiceUnavailable)
		return
	}
	s.filesPresign.Handle(w, r)
}

func (s *Server) DownloadFile(w http.ResponseWriter, r *http.Request, fileID apiTypes.UUID) {
	if s.filesDownload == nil {
		http.Error(w, "download not available", http.StatusServiceUnavailable)
		return
	}
	s.filesDownload.Handle(w, r, uuid.UUID(fileID))
}
