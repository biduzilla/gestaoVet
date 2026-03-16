package models

type BaseModelCnpj struct {
	BaseModel
	Cnpj string `db:"cnpj"`
}
