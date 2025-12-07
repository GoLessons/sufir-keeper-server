package api

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"github.com/google/uuid"
	apiTypes "github.com/oapi-codegen/runtime/types"
	"go.uber.org/zap"

	authhandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/auth"
	itemshandler "github.com/GoLessons/sufir-keeper-server/internal/app/handler/items"
	"github.com/GoLessons/sufir-keeper-server/internal/crypto/keyencrypt"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type ServerDependencies struct {
	DatabaseClient         interface{}
	Logger                 *zap.Logger
	TokenAuth              *jwtauth.JWTAuth
	UsersRepository        *repository.UserRepository
	ItemsRepository        *repository.ItemRepository
	KEKProvider            interface{}
	AccessTokenTTLSeconds  int
	RefreshTokenTTLSeconds int
}

type Server struct {
	Unimplemented
	kek      keyencrypt.Provider
	users    *repository.UserRepository
	items    *repository.ItemRepository
	login    *authhandler.LoginHandler
	refresh  *authhandler.RefreshHandler
	register *authhandler.RegisterHandler
	logout   *authhandler.LogoutHandler
	itemsH   *itemshandler.Handler
	deps     ServerDependencies
}

func NewServer(deps ServerDependencies) *Server {
	users := deps.UsersRepository
	items := deps.ItemsRepository
	var kekProvider keyencrypt.Provider
	if p, ok := deps.KEKProvider.(keyencrypt.Provider); ok {
		kekProvider = p
	}
	return &Server{
		deps:     deps,
		users:    users,
		items:    items,
		login:    authhandler.NewLoginHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		refresh:  authhandler.NewRefreshHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		register: authhandler.NewRegisterHandler(users),
		logout:   authhandler.NewLogoutHandler(users, deps.TokenAuth),
		itemsH:   itemshandler.NewHandler(items, kekProvider, deps.TokenAuth),
		kek:      kekProvider,
	}
}

func (s *Server) LogoutUser(w http.ResponseWriter, r *http.Request)   { s.logout.Handle(w, r) }
func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) { s.refresh.Handle(w, r) }
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request)    { s.login.Handle(w, r) }
func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) { s.register.Handle(w, r) }

func (s *Server) CreateItem(w http.ResponseWriter, r *http.Request) { s.itemsH.HandleCreate(w, r) }

func (s *Server) GetItems(w http.ResponseWriter, r *http.Request, params GetItemsParams) {
	var fType *string
	if params.Type != nil {
		v := string(*params.Type)
		fType = &v
	}
	lim := s.defaultLimit(params.Limit)
	off := s.defaultOffset(params.Offset)
	s.itemsH.HandleList(w, r, fType, params.S, lim, off)
}

func (s *Server) GetItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	s.itemsH.HandleGet(w, r, uuid.UUID(id))
}

func (s *Server) UpdateItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	s.itemsH.HandleUpdate(w, r, uuid.UUID(id))
}

func (s *Server) DeleteItem(w http.ResponseWriter, r *http.Request, id apiTypes.UUID) {
	s.itemsH.HandleDelete(w, r, uuid.UUID(id))
}

func (s *Server) defaultLimit(v *int) int {
	if v == nil || *v <= 0 {
		return 20
	}
	if *v > 100 {
		return 100
	}
	return *v
}

func (s *Server) defaultOffset(v *int) int {
	if v == nil || *v < 0 {
		return 0
	}
	return *v
}

// item response construction moved to app/handler
