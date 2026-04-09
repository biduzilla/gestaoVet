package auth

import (
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type authService struct {
	usuarioService usuario.UsuarioService
	config         config.Config
}

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type AuthService interface {
	Login(
		v *validator.Validator,
		email, password string,
	) (string, string, uuid.UUID, error)

	ExtractUsername(tokenString string) (string, error)
	RefreshToken(refreshToken string) (string, error)
}

func NewService(
	usuarioService usuario.UsuarioService,
	config config.Config,
) *authService {
	return &authService{
		usuarioService: usuarioService,
		config:         config,
	}
}

func (s *authService) Login(
	v *validator.Validator,
	email, password string,
) (string, string, uuid.UUID, error) {
	usuario.ValidatePasswordPlaintext(v, password)

	if !v.Valid() {
		return "", "", uuid.Nil, errors.ErrInvalidData
	}

	user, err := s.usuarioService.FindByEmail(email, v)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	if !user.IsAtivo {
		return "", "", uuid.Nil, errors.ErrInactiveAccount
	}

	match, err := user.Senha.Matches(password)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	if !match {
		return "", "", uuid.Nil, errors.ErrInvalidCredentials
	}

	token, err := s.createAccessToken(user.Email)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	refreshToken, err := s.createRefreshToken(user.Email)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	return token, refreshToken, user.ID, nil
}

func (s *authService) createAccessToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"type":     TokenTypeAccess,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
	tokenStr, err := token.SignedString([]byte(s.config.Security.SecretKey))

	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (s *authService) createRefreshToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"type":     TokenTypeRefresh,
			"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
		})

	return token.SignedString([]byte(s.config.Security.SecretKey))
}

func (s *authService) ExtractUsername(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		return []byte(s.config.Security.SecretKey), nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", nil
	}

	return username, nil
}

func (s *authService) RefreshToken(refreshToken string) (string, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (any, error) {
		return []byte(s.config.Security.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return "", errors.ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.ErrInvalidCredentials
	}

	if claims["type"] != "refresh" {
		return "", errors.ErrInvalidCredentials
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", errors.ErrInvalidCredentials
	}

	return s.createAccessToken(username)
}
