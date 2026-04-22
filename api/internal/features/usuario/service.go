package usuario

import (
	"context"
	"database/sql"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/validator"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type usuarioService struct {
	repository UsuarioRepository
	db         *sql.DB
}

type UsuarioService interface {
	FindByEmail(
		ctx context.Context,
		email string,
	) (*Usuario, error)

	FindByID(
		ctx context.Context,
		ID uuid.UUID,
	) (*Usuario, error)

	FindAll(
		ctx context.Context,
		nome, telefone, email, cnpj string,
		f filters.Filters,
	) ([]*Usuario, filters.Metadata, error)

	Save(
		ctx context.Context,
		model *Usuario,
		tx *sql.Tx,
	) error

	Update(
		ctx context.Context,
		model *Usuario,
	) error

	UpdateRoles(
		ctx context.Context,
		userID uuid.UUID,
		roles []int32,
	) error

	UpdateSenha(
		ctx context.Context,
		userID uuid.UUID,
		senha string,
	) error

	Delete(
		ctx context.Context,
		id uuid.UUID,
	) error
}

func NewService(
	repository UsuarioRepository,
	db *sql.DB,
) *usuarioService {
	return &usuarioService{
		repository: repository,
		db:         db,
	}
}

func (s *usuarioService) FindByEmail(
	ctx context.Context,
	email string,
) (*Usuario, error) {
	v := validator.New()
	if ValidateEmail(v, email); !v.Valid() {
		return nil, errors.ErrInvalidData
	}
	return s.repository.FindByEmail(ctx, email)
}

func (s *usuarioService) FindByID(
	ctx context.Context,
	ID uuid.UUID,
) (*Usuario, error) {
	return s.repository.FindByID(ctx, ID)
}

func (s *usuarioService) FindAll(
	ctx context.Context,
	nome, telefone, email, cnpj string,
	f filters.Filters,
) ([]*Usuario, filters.Metadata, error) {
	return s.repository.FindAll(ctx, nome, telefone, email, cnpj, f)
}

func (s *usuarioService) Save(
	ctx context.Context,
	model *Usuario,
	tx *sql.Tx,
) error {
	saveLogic := func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return errors.NewValidationError(v.Errors)
		}

		model.Roles = append(model.Roles, int32(interfaces.ROLE_RECEPTIONIST))
		return s.repository.Insert(ctx, tx, model)
	}

	if tx != nil {
		return saveLogic(tx)
	}

	return utils.RunInTx(s.db, saveLogic)
}

func (s *usuarioService) UpdateRoles(
	ctx context.Context,
	userID uuid.UUID,
	roles []int32,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		user, err := s.FindByID(ctx, userID)
		if err != nil {
			return err
		}

		user.Roles = roles
		user.SetRolesReplace()
		return s.Update(ctx, user)
	})
}

func (s *usuarioService) Update(
	ctx context.Context,
	model *Usuario,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return errors.NewValidationError(v.Errors)
		}

		return s.repository.Update(ctx, tx, model)
	})
}

func (s *usuarioService) UpdateSenha(
	ctx context.Context,
	userID uuid.UUID,
	senha string,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		user, err := s.FindByID(ctx, userID)
		if err != nil {
			return err
		}
		user.Senha.Set(senha)
		return s.repository.UpdateSenha(ctx, tx, user)
	})
}

func (s *usuarioService) Delete(
	ctx context.Context,
	id uuid.UUID,
) error {
	return utils.RunInTx(s.db, func(tx *sql.Tx) error {
		return s.repository.Delete(ctx, tx, id)
	})
}
