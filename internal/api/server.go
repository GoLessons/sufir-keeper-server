package api

import (
	"net/http"

	"github.com/go-chi/jwtauth/v5"
	"go.uber.org/zap"

	apiHandlers "github.com/GoLessons/sufir-keeper-server/internal/api/handler"
	"github.com/GoLessons/sufir-keeper-server/internal/repository"
)

type ServerDependencies struct {
	DatabaseClient         interface{}
	Logger                 *zap.Logger
	TokenAuth              *jwtauth.JWTAuth
	UsersRepository        *repository.UserRepository
	AccessTokenTTLSeconds  int
	RefreshTokenTTLSeconds int
}

type Server struct {
	Unimplemented
	users    *repository.UserRepository
	login    *apiHandlers.LoginHandler
	refresh  *apiHandlers.RefreshHandler
	register *apiHandlers.RegisterHandler
	logout   *apiHandlers.LogoutHandler
	deps     ServerDependencies
}

func NewServer(deps ServerDependencies) *Server {
	users := deps.UsersRepository
	return &Server{
		deps:     deps,
		users:    users,
		login:    apiHandlers.NewLoginHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		refresh:  apiHandlers.NewRefreshHandler(users, deps.TokenAuth, deps.AccessTokenTTLSeconds, deps.RefreshTokenTTLSeconds),
		register: apiHandlers.NewRegisterHandler(users),
		logout:   apiHandlers.NewLogoutHandler(users, deps.TokenAuth),
	}
}

func (s *Server) LogoutUser(w http.ResponseWriter, r *http.Request)   { s.logout.Handle(w, r) }
func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) { s.refresh.Handle(w, r) }
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request)    { s.login.Handle(w, r) }
func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) { s.register.Handle(w, r) }
