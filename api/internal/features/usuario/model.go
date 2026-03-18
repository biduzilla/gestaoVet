package usuario

import (
	"errors"
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"
	"regexp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Usuario struct {
	models.BaseModelCnpj
	ID       uuid.UUID `db:"id"`
	Nome     string    `db:"nome"`
	Telefone string    `db:"telefone"`
	Email    string    `db:"email"`
	Senha    password  `db:"-"`
	IsAtivo  bool      `db:"is_ativo"`
}

type UsuarioDTO struct {
	ID       *uuid.UUID `json:"id"`
	Nome     *string    `json:"nome"`
	Telefone *string    `json:"telefone"`
	Email    *string    `json:"email"`
	Cnpj     *string    `json:"cnpj"`
	Senha    *string    `json:"-"`
}

func (u *Usuario) IsAnonymous() bool {
	return false
}

type password struct {
	Plaintext *string
	Hash      []byte `db:"password_hash"`
}

func (m Usuario) toDTO() *UsuarioDTO {
	return &UsuarioDTO{
		ID:       &m.ID,
		Nome:     &m.Nome,
		Telefone: &m.Telefone,
		Email:    &m.Email,
	}
}

func (d UsuarioDTO) toModel() (*Usuario, error) {
	var model Usuario

	if d.ID != nil {
		model.ID = *d.ID
	}

	if d.Nome != nil {
		model.Nome = *d.Nome
	}

	if d.Telefone != nil {
		model.Telefone = *d.Telefone
	}

	if d.Email != nil {
		model.Email = *d.Email
	}

	if d.Senha != nil {
		err := model.Senha.Set(*d.Senha)
		if err != nil {
			return nil, err
		}
	}

	return &model, nil
}

func (u *Usuario) Validate(v *validator.Validator) {
	v.Check(u.Nome != "", "nome", "must be provided")
	v.Check(len(u.Nome) >= 3, "nome", "must be at least 3 characters long")
	v.Check(len(u.Nome) <= 100, "nome", "must not be more than 100 characters long")
	v.Check(u.Telefone != "", "telefone", "must be provided")
	v.Check(ValidateTelefone(u.Telefone), "telefone", "invalid telephone format")
	v.Check(u.Email != "", "email", "must be provided")
	v.Check(validator.Matches(u.Email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidateTelefone(telefone string) bool {
	telefone = regexp.MustCompile(`[^\d]`).ReplaceAllString(telefone, "")

	switch len(telefone) {
	case 8, 9, 10, 11:
		return true
	default:
		return false
	}
}

func (p *password) Set(plaintextPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plaintextPassword), 12)
	if err != nil {
		return err
	}

	p.Plaintext = &plaintextPassword
	p.Hash = hash
	return nil
}

func (p *password) Matches(plaintextPassword string) (bool, error) {
	err := bcrypt.CompareHashAndPassword(p.Hash, []byte(plaintextPassword))
	if err != nil {
		switch {
		case errors.Is(err, bcrypt.ErrMismatchedHashAndPassword):
			return false, nil
		default:
			return false, err
		}
	}
	return true, nil
}

func ValidateEmail(v *validator.Validator, email string) {
	v.Check(email != "", "email", "must be provided")
	v.Check(validator.Matches(email, validator.EmailRX), "email", "must be a valid email address")
}

func ValidatePasswordPlaintext(v *validator.Validator, password string) {
	v.Check(password != "", "password", "must be provided")
	v.Check(len(password) >= 8, "password", "must be at least 8 bytes long")
	v.Check(len(password) <= 72, "password", "must not be more than 72 bytes long")
}

func (u *Usuario) GetID() uuid.UUID {
	return u.ID
}

func (u *Usuario) GetCNPJ() string {
	return u.Cnpj
}

func (u *Usuario) GetIsAtivo() bool {
	return u.IsAtivo
}
