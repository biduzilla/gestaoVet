package auth

import (
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type authService struct {
	usuarioService usuario.UsuarioService
	config         config.Config
}

type AuthService interface {
	Login(
		v *validator.Validator,
		email, password string,
	) (string, error)

	ExtractUsername(tokenString string) (string, error)
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
) (string, error) {
	usuario.ValidatePasswordPlaintext(v, password)

	if !v.Valid() {
		return "", errors.ErrInvalidData
	}

	user, err := s.usuarioService.FindByEmail(email, v)
	if err != nil {
		return "", err
	}

	if !user.IsAtivo {
		return "", errors.ErrInactiveAccount
	}

	match, err := user.Senha.Matches(password)
	if err != nil {
		return "", err
	}

	if !match {
		return "", errors.ErrInvalidCredentials
	}

	token, err := s.createToken(user.Email)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *authService) createToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"username": username,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
		})
	tokenStr, err := token.SignedString([]byte(s.config.Security.SecretKey))

	if err != nil {
		return "", err
	}

	return tokenStr, nil
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
