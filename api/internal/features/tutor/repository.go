package tutor

import (
	"context"
	"database/sql"
	"gestaoVet/internal/core/contexts"
	e "gestaoVet/internal/core/domain/errors"
	"gestaoVet/internal/core/filters"
	"gestaoVet/internal/core/jsonlog"
	"gestaoVet/internal/core/repository"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type tutorRepository struct {
	db             *sql.DB
	logger         jsonlog.Logger
	baseRepository repository.BaseRepository[Tutor]
}
type TutorRepository interface {
	FindByID(
		ctx context.Context,
		ID uuid.UUID,
	) (*Tutor, error)

	FindAll(
		ctx context.Context,
		search string,
		f filters.Filters,
	) ([]*Tutor, filters.Metadata, error)

	Insert(
		ctx context.Context,
		tx *sql.Tx,
		model *Tutor,
	) error

	Update(
		ctx context.Context,
		tx *sql.Tx,
		model *Tutor,
	) error

	DeleteByID(
		ctx context.Context,
		ID uuid.UUID,
		tx *sql.Tx,
	) error
}

func NewRepository(
	db *sql.DB,
	logger jsonlog.Logger,
) TutorRepository {
	return &tutorRepository{
		db:             db,
		logger:         logger,
		baseRepository: repository.NewBaseRepository[Tutor](db, logger, "tutores", "t"),
	}
}

func parseTutorConstraintError(err error) error {
	if pqErr, ok := err.(*pq.Error); ok {
		switch pqErr.Constraint {
		case "tutor_identidade_key":
			return e.ValidationAlreadyExists("identidade")
		case "tutor_cpf_key":
			return e.ValidationAlreadyExists("cpf")
		case "tutor_celular_key":
			return e.ValidationAlreadyExists("celular")
		}
	}
	return err
}

func (r *tutorRepository) FindByID(
	ctx context.Context,
	ID uuid.UUID,
) (*Tutor, error) {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"id":   ID,
		"cnpj": user.GetCNPJ(),
	}

	return r.baseRepository.FindOne(ctx, `
		t.id = :id 
		and t.cnpj = :cnpj
	`, params)
}

func (r *tutorRepository) FindAll(
	ctx context.Context,
	search string,
	f filters.Filters,
) ([]*Tutor, filters.Metadata, error) {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"search": search,
		"cnpj":   user.GetCNPJ(),
	}

	query := `
			(
				(:search is null or :search = '')
				or to_tsvector('simple', t.nome) @@ plainto_tsquery('simple', :search)
				or to_tsvector('simple', t.cpf) @@ plainto_tsquery('simple', :search)
				or to_tsvector('simple', t.telefone1) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.telefone2) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.email1) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.email2) @@ plainto_tsquery('simple', :search) 
				or to_tsvector('simple', t.identidade) @@ plainto_tsquery('simple', :search)
				)
			and t.cnpj = :cnpj
	`

	return r.baseRepository.FindWithFilters(ctx, f, query, params)
}

func (r *tutorRepository) Insert(
	ctx context.Context,
	tx *sql.Tx,
	model *Tutor,
) error {
	user := contexts.ContextGetUser(ctx)

	err := r.baseRepository.Insert(
		ctx,
		tx,
		model,
		repository.WithExtraWhere("", map[string]any{
			"cnpj":      user.GetCNPJ(),
			"createdBy": user.GetID(),
		}),
	)

	if err != nil {
		return parseTutorConstraintError(err)
	}
	return nil
	// query := `
	// insert into tutores (
	// 	nome,
	// 	celular,
	// 	sexo,
	// 	nascimento,
	// 	identidade,
	// 	cpf,
	// 	observacoes,
	// 	cep,
	// 	endereco,
	// 	bairro,
	// 	cidade,
	// 	telefone1,
	// 	telefone2,
	// 	email1,
	// 	email2,
	// 	numero,
	// 	complemento,
	// 	estado,
	// 	cnpj,
	// 	created_by
	// )
	// values (
	// 	:nome,
	// 	:celular,
	// 	:sexo,
	// 	:nascimento,
	// 	:identidade,
	// 	:cpf,
	// 	:observacoes,
	// 	:cep,
	// 	:endereco,
	// 	:bairro,
	// 	:cidade,
	// 	:telefone1,
	// 	:telefone2,
	// 	:email1,
	// 	:email2,
	// 	:numero,
	// 	:complemento,
	// 	:estado,
	// 	:cnpj,
	// 	:createdBy
	// )
	// returning
	// 	id,
	// 	created_at,
	// 	version
	// `

	// params := map[string]any{
	// 	"nome":        model.Nome,
	// 	"celular":     model.Celular,
	// 	"sexo":        model.Sexo,
	// 	"nascimento":  model.Nascimento,
	// 	"identidade":  model.Identidade,
	// 	"cpf":         model.CPF,
	// 	"observacoes": model.Observacoes,
	// 	"cep":         model.CEP,
	// 	"endereco":    model.Endereco,
	// 	"bairro":      model.Bairro,
	// 	"cidade":      model.Cidade,
	// 	"telefone1":   model.Telefone1,
	// 	"telefone2":   model.Telefone2,
	// 	"email1":      model.Email1,
	// 	"email2":      model.Email2,
	// 	"numero":      model.Numero,
	// 	"complemento": model.Complemento,
	// 	"estado":      model.Estado,
	// 	"cnpj":        model.Cnpj,
	// 	"createdBy":   user.GetID(),
	// }

	// query, args := repository.NamedQuery(query, params)

	// err := tx.QueryRowContext(ctx, query, args...).Scan(
	// 	&model.ID,
	// 	&model.CreatedAt,
	// 	&model.Version,
	// )

	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return e.ErrRecordNotFound
	// 	}

	// 	return parseTutorConstraintError(err)
	// }

	// return nil
}

func (r *tutorRepository) Update(
	ctx context.Context,
	tx *sql.Tx,
	model *Tutor,
) error {
	user := contexts.ContextGetUser(ctx)

	err := r.baseRepository.Update(
		ctx,
		tx,
		model,
		model.ID,
		repository.WithExtraWhere("and cnpj = :cnpj", map[string]any{
			"cnpj":      user.GetCNPJ(),
			"updatedBy": user.GetID(),
		}),
	)

	if err != nil {
		return parseTutorConstraintError(err)
	}
	return nil
	// query := `
	// update tutores
	// set
	// 	nome = :nome,
	// 	celular = :celular,
	// 	sexo = :sexo,
	// 	nascimento = :nascimento,
	// 	identidade = :identidade,
	// 	cpf = :cpf,
	// 	observacoes = :observacoes,
	// 	cep = :cep,
	// 	endereco = :endereco,
	// 	bairro = :bairro,
	// 	cidade = :cidade,
	// 	telefone1 = :telefone1,
	// 	telefone2 = :telefone2,
	// 	email1 = :email1,
	// 	email2 = :email2,
	// 	numero = :numero,
	// 	complemento = :complemento,
	// 	estado = :estado,
	// 	updated_at = now(),
	// 	updated_by = :ID,
	// 	version = tutores.version + 1
	// where
	// 	id = :tutorId
	// 	and cnpj = :cnpj
	// 	and version = :version
	// 	and deleted = false
	// returning
	// 	version
	// `

	// params := map[string]any{
	// 	"nome":        model.Nome,
	// 	"celular":     model.Celular,
	// 	"sexo":        model.Sexo,
	// 	"nascimento":  model.Nascimento,
	// 	"identidade":  model.Identidade,
	// 	"cpf":         model.CPF,
	// 	"observacoes": model.Observacoes,
	// 	"cep":         model.CEP,
	// 	"endereco":    model.Endereco,
	// 	"bairro":      model.Bairro,
	// 	"cidade":      model.Cidade,
	// 	"telefone1":   model.Telefone1,
	// 	"telefone2":   model.Telefone2,
	// 	"email1":      model.Email1,
	// 	"email2":      model.Email2,
	// 	"numero":      model.Numero,
	// 	"complemento": model.Complemento,
	// 	"estado":      model.Estado,
	// 	"cnpj":        user.GetCNPJ(),
	// 	"ID":          user.GetID(),
	// 	"version":     model.Version,
	// 	"tutorId":     model.ID,
	// }

	// query, args := repository.NamedQuery(query, params)

	// r.logger.PrintInfo(utils.MinifySQL(query), nil)

	// err := tx.QueryRowContext(ctx, query, args...).Scan(
	// 	&model.Version,
	// )

	// if err != nil {
	// 	if errors.Is(err, sql.ErrNoRows) {
	// 		return e.ErrEditConflict
	// 	}

	// 	return parseTutorConstraintError(err)
	// }

	// return nil
}

func (r *tutorRepository) DeleteByID(
	ctx context.Context,
	ID uuid.UUID,
	tx *sql.Tx,
) error {
	user := contexts.ContextGetUser(ctx)
	params := map[string]any{
		"id":     ID,
		"userID": user.GetID(),
		"cnpj":   user.GetCNPJ(),
	}

	query := `
		id = :id
		and cnpj = :cnpj
	`
	return r.baseRepository.DeleteByQuery(ctx, tx, query, params)
}
