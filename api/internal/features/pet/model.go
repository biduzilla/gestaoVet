package pet

import (
	"gestaoVet/internal/core/domain/models"
	"gestaoVet/internal/core/validator"
	"gestaoVet/internal/features/tutor"
	"gestaoVet/utils"

	"github.com/google/uuid"
)

type Pet struct {
	models.BaseModel
	ID          uuid.UUID    `db:"id" repo:"auto"`
	Nome        string       `db:"nome" repo:"insert,update"`
	Especie     string       `db:"especie" repo:"insert,update"`
	Raca        string       `db:"raca" repo:"insert,update"`
	Nascimento  string       `db:"nascimento" repo:"insert,update"`
	Sexo        string       `db:"sexo" repo:"insert,update"`
	Pelo        string       `db:"pelo" repo:"insert,update"`
	Cor         string       `db:"cor" repo:"insert,update"`
	Peso        float64      `db:"peso" repo:"insert,update"`
	Castrado    bool         `db:"castrado" repo:"insert,update"`
	Microchip   string       `db:"microchip" repo:"insert,update"`
	Restricoes  string       `db:"restricoes" repo:"insert,update"`
	Observacoes string       `db:"observacoes" repo:"insert,update"`
	Tutor       *tutor.Tutor `db:"-" dto:"User"`
}

type PetDTO struct {
	ID          *uuid.UUID      `json:"id"`
	Nome        *string         `json:"nome"`
	Especie     *string         `json:"especie"`
	Raca        *string         `json:"raca"`
	Nascimento  *string         `json:"nascimento"`
	Sexo        *string         `json:"sexo"`
	Pelo        *string         `json:"pelo"`
	Cor         *string         `json:"cor"`
	Peso        *float64        `json:"peso"`
	Castrado    *bool           `json:"castrado"`
	Microchip   *string         `json:"microchip"`
	Restricoes  *string         `json:"restricoes"`
	Observacoes *string         `json:"observacoes"`
	Tutor       *tutor.TutorDTO `json:"tutor,omitempty"`
	Version     *int            `json:"version"`
}

func (p Pet) toDTO() *PetDTO {
	dto := &PetDTO{
		ID:          &p.ID,
		Nome:        &p.Nome,
		Especie:     &p.Especie,
		Raca:        &p.Raca,
		Nascimento:  &p.Nascimento,
		Sexo:        &p.Sexo,
		Pelo:        &p.Pelo,
		Cor:         &p.Cor,
		Peso:        &p.Peso,
		Castrado:    &p.Castrado,
		Microchip:   &p.Microchip,
		Restricoes:  &p.Restricoes,
		Observacoes: &p.Observacoes,
		Version:     &p.Version,
	}

	if p.Tutor != nil {
		dto.Tutor = p.Tutor.ToDTO()
	}

	return dto
}

func (d PetDTO) ToModel() *Pet {
	model := &Pet{}

	if d.ID != nil {
		model.ID = *d.ID
	}
	if d.Nome != nil {
		model.Nome = *d.Nome
	}
	if d.Especie != nil {
		model.Especie = *d.Especie
	}
	if d.Raca != nil {
		model.Raca = *d.Raca
	}
	if d.Nascimento != nil {
		model.Nascimento = *d.Nascimento
	}
	if d.Sexo != nil {
		model.Sexo = *d.Sexo
	}
	if d.Pelo != nil {
		model.Pelo = *d.Pelo
	}
	if d.Cor != nil {
		model.Cor = *d.Cor
	}
	if d.Peso != nil {
		model.Peso = *d.Peso
	}
	if d.Castrado != nil {
		model.Castrado = *d.Castrado
	}
	if d.Microchip != nil {
		model.Microchip = *d.Microchip
	}
	if d.Restricoes != nil {
		model.Restricoes = *d.Restricoes
	}
	if d.Observacoes != nil {
		model.Observacoes = *d.Observacoes
	}
	if d.Version != nil {
		model.Version = *d.Version
	}

	if d.Tutor != nil {
		model.Tutor = d.Tutor.ToModel()
	}

	return model
}

func (p *Pet) Validate(v *validator.Validator) {
	v.Check(p.Nome != "", "nome", "must be provided")
	v.Check(len(p.Nome) >= 2, "nome", "must be at least 2 characters long")
	v.Check(len(p.Nome) <= 100, "nome", "must not be more than 100 characters long")

	v.Check(p.Especie != "", "especie", "must be provided")
	v.Check(p.Especie == "Cão" || p.Especie == "Gato" || p.Especie == "Outro", "especie", "must be 'Cão', 'Gato' or 'Outro'")

	v.Check(p.Raca != "", "raca", "must be provided")
	v.Check(len(p.Raca) <= 50, "raca", "must not be more than 50 characters long")

	v.Check(p.Nascimento != "", "nascimento", "must be provided")
	v.Check(utils.ValidateDate(p.Nascimento), "nascimento", "invalid date format (DD/MM/YYYY)")

	v.Check(p.Sexo != "", "sexo", "must be provided")
	v.Check(p.Sexo == "M" || p.Sexo == "F", "sexo", "must be 'M' or 'F'")

	if p.Pelo != "" {
		v.Check(p.Pelo == "Curto" || p.Pelo == "Médio" || p.Pelo == "Longo", "pelo", "must be 'Curto', 'Médio' or 'Longo'")
	}

	if p.Cor != "" {
		v.Check(len(p.Cor) <= 50, "cor", "must not be more than 50 characters long")
	}

	v.Check(p.Peso > 0, "peso", "must be greater than 0")
	v.Check(p.Peso <= 200, "peso", "must not be more than 200 kg")

	if p.Microchip != "" {
		v.Check(len(p.Microchip) <= 50, "microchip", "must not be more than 50 characters long")
	}

	v.Check(p.Tutor != nil, "tutor", "must be provided")

	if p.Restricoes != "" {
		v.Check(len(p.Restricoes) <= 500, "restricoes", "must not be more than 500 characters long")
	}
	if p.Observacoes != "" {
		v.Check(len(p.Observacoes) <= 1000, "observacoes", "must not be more than 1000 characters long")
	}
}

func ValidateNome(v *validator.Validator, nome string) {
	v.Check(nome != "", "nome", "must be provided")
	v.Check(len(nome) >= 2, "nome", "must be at least 2 characters long")
	v.Check(len(nome) <= 100, "nome", "must not be more than 100 characters long")
}

func ValidateDataNascimento(v *validator.Validator, data string) {
	v.Check(data != "", "nascimento", "must be provided")
	v.Check(utils.ValidateDate(data), "nascimento", "invalid date format (DD/MM/YYYY)")
}

func ValidatePeso(v *validator.Validator, peso float64) {
	v.Check(peso > 0, "peso", "must be greater than 0")
	v.Check(peso <= 200, "peso", "must not be more than 200 kg")
}
