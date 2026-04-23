package empresa

import (
	"context"
	"database/sql"
	"gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/interfaces"
	"gestaoVet/internal/core/transaction"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/usuario"
)

type empresaService struct {
	repository     EmpresaRepository
	usuarioService usuario.UsuarioService
	tx             transaction.Manager
}

func NewService(
	usuarioService usuario.UsuarioService,
	repository EmpresaRepository,
	tx transaction.Manager,
) *empresaService {
	return &empresaService{
		usuarioService: usuarioService,
		repository:     repository,
		tx:             tx,
	}
}

type EmpresaService interface {
	FindAll(
		ctx context.Context,
		cnpj, nomeFantasia, razaoSocial, email string,
		f filters.Filters,
	) ([]*Empresa, filters.Metadata, error)
	FindByCnpj(ctx context.Context, cnpj string) (*Empresa, error)
	Save(ctx context.Context,
		model *Empresa,
	) error
	Update(ctx context.Context,
		model *Empresa,
	) error
	Delete(ctx context.Context) error
}

func (s *empresaService) FindAll(
	ctx context.Context,
	cnpj, nomeFantasia, razaoSocial, email string,
	f filters.Filters,
) ([]*Empresa, filters.Metadata, error) {
	return s.repository.FindAll(ctx, cnpj, nomeFantasia, razaoSocial, email, f)
}

func (s *empresaService) Save(
	ctx context.Context,
	model *Empresa,
) error {
	return s.tx.RunInTx(ctx, func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return errors.NewValidationError(v.Errors)
		}

		model.IsAtivo = true
		err := s.repository.Insert(ctx, tx, model)
		if err != nil {
			return err
		}

		return s.createUserAdmin(ctx, model, tx)
	})
}

func (s *empresaService) Update(
	ctx context.Context,
	model *Empresa,
) error {
	return s.tx.RunInTx(ctx, func(tx *sql.Tx) error {
		v := validator.New()
		if model.Validate(v); !v.Valid() {
			return errors.NewValidationError(v.Errors)
		}

		return s.repository.Update(ctx, tx, model)
	})
}

func (s *empresaService) FindByCnpj(ctx context.Context, cnpj string) (*Empresa, error) {
	return s.repository.FindByCnpj(ctx, cnpj)
}

func (s *empresaService) Delete(ctx context.Context) error {
	return s.tx.RunInTx(ctx, func(tx *sql.Tx) error {
		return s.repository.Delete(ctx, tx)
	})
}

func (s *empresaService) createUserAdmin(
	ctx context.Context,
	model *Empresa,
	tx *sql.Tx,
) error {
	var user = usuario.Usuario{
		Nome:     model.RazaoSocial,
		Telefone: model.Telefone,
		Email:    model.Email,
		BaseModelCnpj: models.BaseModelCnpj{
			Cnpj: model.Cnpj,
		},
		IsAtivo: true,
		Roles:   []int32{int32(interfaces.ROLE_ADMIN)},
	}
	user.Senha.Set(model.Cnpj)
	return s.usuarioService.Save(ctx, &user, tx)
}
