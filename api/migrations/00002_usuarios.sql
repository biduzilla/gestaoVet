-- +goose Up
CREATE TABLE IF NOT EXISTS usuarios (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    nome varchar(100) NOT NULL,
    telefone varchar(20) NOT NULL,
    email varchar(255) NOT NULL,
    password_hash bytea NOT NULL,
    cnpj varchar(14) NOT NULL REFERENCES empresas(cnpj),
    is_ativo boolean NOT NULL DEFAULT true,
    deleted boolean NOT NULL DEFAULT false,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid,
    updated_at timestamptz,
    updated_by uuid
);

CREATE UNIQUE INDEX user_email_key 
ON usuarios (email) 
WHERE deleted = false;

CREATE UNIQUE INDEX user_telefone_key 
ON usuarios (telefone) 
WHERE deleted = false;

CREATE INDEX idx_usuarios_cnpj 
ON usuarios(cnpj) 
WHERE deleted = false;

CREATE INDEX idx_usuarios_nome 
ON usuarios USING GIN (to_tsvector('simple', nome));

CREATE INDEX idx_usuarios_email 
ON usuarios USING GIN (to_tsvector('simple', email));

CREATE INDEX idx_usuarios_telefone 
ON usuarios USING GIN (to_tsvector('simple', telefone));

CREATE INDEX idx_usuarios_deleted 
ON usuarios(deleted);

CREATE INDEX idx_usuarios_cnpj_deleted 
ON usuarios(cnpj, deleted);

-- +goose Down
DROP TABLE IF EXISTS usuarios CASCADE;