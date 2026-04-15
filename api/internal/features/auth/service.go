package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"gestaoVet/internal/core/config"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
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

	ExtractUsername(tokenString string) (string, error)
	ExtractUserID(tokenString string) (uuid.UUID, error)
	ExtractRoles(tokenString string) ([]int32, error)
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
		return nil, fmt.Errorf("falha ao carregar chaves RSA: %w", err)
	}

	return service, nil
}

func (s *authService) loadKeys() error {
	privateKeyData, err := os.ReadFile(s.config.Security.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("erro ao ler chave privada: %w", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("falha ao decodificar PEM da chave privada")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return fmt.Errorf("falha ao parsear chave privada: %v / %v", err, err2)
		}
		var ok bool
		privateKey, ok = key.(*rsa.PrivateKey)
		if !ok {
			return fmt.Errorf("chave não é RSA")
		}
	}
	s.privateKey = privateKey

	publicKeyData, err := os.ReadFile(s.config.Security.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("erro ao ler chave pública: %w", err)
	}

	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		return fmt.Errorf("falha ao decodificar PEM da chave pública")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("falha ao parsear chave pública: %w", err)
	}

	publicKey, ok := publicKeyInterface.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("chave pública não é RSA")
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

	token, err := s.createAccessToken(user.Email, user.ID.String(), user.Roles)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	refreshToken, err := s.createRefreshToken(user.Email, user.ID.String(), user.Roles)
	if err != nil {
		return "", "", uuid.Nil, err
	}

	return token, refreshToken, user.ID, nil
}

func (s *authService) createAccessToken(username string, userID string, roles []int32) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256,
		jwt.MapClaims{
			"username": username,
			"user_id":  userID,
			"roles":    roles,
			"type":     TokenTypeAccess,
			"exp":      time.Now().Add(time.Hour * 24).Unix(),
			"iat":      time.Now().Unix(),
		})
	tokenStr, err := token.SignedString(s.privateKey)

	if err != nil {
		return "", err
	}

	return tokenStr, nil
}

func (s *authService) createRefreshToken(username string, userID string, roles []int32) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodRS256,
		jwt.MapClaims{
			"username": username,
			"user_id":  userID,
			"roles":    roles,
			"type":     TokenTypeRefresh,
			"exp":      time.Now().Add(7 * 24 * time.Hour).Unix(),
			"iat":      time.Now().Unix(),
		})

	return token.SignedString(s.privateKey)
}

func (s *authService) ExtractUsername(tokenString string) (string, error) {
	claims, ok, err := s.extractClaims(tokenString)
	if !ok {
		return "", nil
	}

	if err != nil {
		return "", err
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", nil
	}

	return username, nil
}

func (s *authService) ExtractUserID(tokenString string) (uuid.UUID, error) {
	claims, ok, err := s.extractClaims(tokenString)

	if err != nil {
		return uuid.Nil, err
	}

	if !ok {
		return uuid.Nil, nil
	}

	userIDStr, ok := claims["user_id"].(string)
	if !ok {
		return uuid.Nil, errors.ErrInvalidCredentials
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("user_id inválido: %w", err)
	}

	return userID, nil
}

func (s *authService) ExtractRoles(tokenString string) ([]int32, error) {
	claims, ok, err := s.extractClaims(tokenString)

	if err != nil {
		return []int32{}, err
	}

	if !ok {
		return []int32{}, nil
	}

	return s.getRolesFromClaims(claims)
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
			return []int32{}, fmt.Errorf("formato de roles inválido")
		}

	}

	return roles, nil
}

func (s *authService) RefreshToken(refreshToken string) (string, error) {
	token, err := s.ValidateToken(refreshToken)
	if err != nil || !token.Valid {
		return "", errors.ErrInvalidCredentials
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.ErrInvalidCredentials
	}

	if claims["type"] != string(TokenTypeRefresh) {
		return "", errors.ErrInvalidCredentials
	}

	username, ok := claims["username"].(string)
	if !ok {
		return "", errors.ErrInvalidCredentials
	}

	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.ErrInvalidCredentials
	}

	roles, err := s.getRolesFromClaims(claims)

	return s.createAccessToken(username, userID, roles)
}

func (s *authService) ValidateToken(tokenString string) (*jwt.Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("método de assinatura inesperado: %v", t.Header["alg"])
		}

		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.ErrInvalidCredentials
	}

	return token, nil
}

func (s *authService) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}
