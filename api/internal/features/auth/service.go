package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"gestaoVet/internal/core/config"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
	"gestaoVet/utils"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type authService struct {
	usuarioService usuario.UsuarioService
	config         config.Config
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
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

	ExtractAuthenticatedUser(tokenString string) (interfaces.User, error)
	RefreshToken(refreshToken string) (string, error)
	ValidateToken(tokenString string) (*jwt.Token, error)
	GetPublicKey() *rsa.PublicKey
}

func NewService(
	usuarioService usuario.UsuarioService,
	config config.Config,
) (AuthService, error) {
	service := &authService{
		usuarioService: usuarioService,
		config:         config,
	}

	if err := service.loadKeys(); err != nil {
		return nil, fmt.Errorf("Failed to load RSA keys.: %w", err)
	}

	return service, nil
}

func (s *authService) loadKeys() error {
	privateKeyData, err := os.ReadFile(s.config.Security.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("Error reading private key.: %w", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("Failure to decode private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return fmt.Errorf("Failed to parse private key.: %v / %v", err, err2)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("The key is not RSA.")
		}
	}
	s.privateKey = privateKey

	publicKeyData, err := os.ReadFile(s.config.Security.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("Error reading public key.: %w", err)
	}

	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		return fmt.Errorf("Failure to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("Failed to parse public key.: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("Public key is not RSA.")
	}
	s.publicKey = publicKey

	return nil
}

func (s *authService) Login(
	v *validator.Validator,
	email, password string,
) (string, string, uuid.UUID, error) {
	usuario.ValidatePasswordPlaintext(v, password)

	if !v.Valid() {
		return "", "", uuid.Nil, e.ErrInvalidData
	}

	user, err := s.usuarioService.FindByEmail(email, v)
	if err != nil {
		if errors.Is(err, e.ErrRecordNotFound) {
			return "", "", uuid.Nil, e.ErrInvalidCredentials
		}
		return "", "", uuid.Nil, err
	}

	if !user.IsAtivo {
		return "", "", uuid.Nil, e.ErrInactiveAccount
	}

	match, err := user.Senha.Matches(password)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	if !match {
		return "", "", uuid.Nil, e.ErrInvalidCredentials
	}

	token, err := s.createAccessToken(user)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	refreshToken, err := s.createRefreshToken(user)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	return token, refreshToken, user.ID, nil
}

func (s *authService) createAccessToken(user interfaces.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256,
		jwt.MapClaims{
			"username": user.GetUsername(),
			"user_id":  user.GetID(),
			"cnpj":     user.GetCNPJ(),
			"is_ativo": user.GetIsAtivo(),
			"roles":    user.GetRoles(),
			"type":     TokenTypeAccess,
			"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
			"iat":      time.Now().Unix(),
		})
	tokenStr, err := token.SignedString(s.privateKey)

	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (s *authService) createRefreshToken(user interfaces.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256,
		jwt.MapClaims{
			"username": user.GetUsername(),
			"user_id":  user.GetID(),
			"cnpj":     user.GetCNPJ(),
			"is_ativo": user.GetIsAtivo(),
			"roles":    user.GetRoles(),
			"type":     TokenTypeRefresh,
			"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
			"iat":      time.Now().Unix(),
		})

	return token.SignedString(s.privateKey)
}

func (s *authService) ExtractAuthenticatedUser(token string) (interfaces.User, error) {
	claims, ok, err := s.extractClaims(token)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return nil, nil
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("Invalid user_id: %w", err)
	}

	username, ok := claims["username"].(string)
	if !ok {
		return nil, nil
	}

	cnpj, ok := claims["cnpj"].(string)
	if !ok {
		return nil, nil
	}

	isAtivo, ok := claims["is_ativo"].(bool)
	if !ok {
		isAtivo = false
	}

	roles, err := s.getRolesFromClaims(claims)
	if err != nil {
		return nil, err
	}

	return models.NewAuthenticatedUser(userID, username, cnpj, isAtivo, utils.ConvertInt32ToRoles(roles)), nil

}

func (s *authService) extractClaims(tokenString string) (jwt.MapClaims, bool, error) {
	token, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, false, err
	}

	if !token.Valid {
		return nil, false, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, false, nil
	}
	return claims, true, nil
}

func (s *authService) getRolesFromClaims(claims jwt.MapClaims) ([]int32, error) {
	var roles []int32
	if rolesInterface, ok := claims["roles"]; ok {
		switch v := rolesInterface.(type) {
		case []any:
			for _, r := range v {
				if num, ok := r.(float64); ok {
					roles = append(roles, int32(num))
				}
			}
		case []float64:
			for _, r := range v {
				roles = append(roles, int32(r))
			}
		default:
			return []int32{}, fmt.Errorf("Invalid role format")
		}

	}

	return roles, nil
}

func (s *authService) RefreshToken(refreshToken string) (string, error) {
	token, err := s.ValidateToken(refreshToken)
	if err != nil || !token.Valid {
		return "", e.ErrInvalidCredentials
	}

	user, err := s.ExtractAuthenticatedUser(refreshToken)
	if err != nil {
		return "", err
	}

	return s.createAccessToken(user)
}

func (s *authService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", t.Header["alg"])
		}

		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, e.ErrInvalidCredentials
	}

	return token, nil
}

func (s *authService) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}
