-- +goose Up
CREATE TABLE IF NOT EXISTS empresas (
    cnpj varchar(14) PRIMARY KEY,
    nome_fantasia varchar(100) NOT NULL,
    telefone varchar(20) NOT NULL,
    razao_social varchar(100) NOT NULL,
    email varchar(255) NOT NULL,
    is_ativo boolean NOT NULL DEFAULT true,
    deleted boolean NOT NULL DEFAULT false,
    version integer NOT NULL DEFAULT 1,
    
    created_at timestamptz NOT NULL DEFAULT now(),
    created_by uuid,
    updated_at timestamptz,
    updated_by uuid
);

CREATE UNIQUE INDEX empresas_email_key 
ON empresas (email) 
WHERE deleted = false;

CREATE UNIQUE INDEX empresas_razao_social_key 
ON empresas (razao_social) 
WHERE deleted = false;

CREATE UNIQUE INDEX empresas_nome_fantasia_key 
ON empresas (nome_fantasia) 
WHERE deleted = false;

CREATE UNIQUE INDEX empresas_nome_telefone_key 
ON empresas (telefone) 
WHERE deleted = false;

CREATE INDEX idx_empresas_cnpj ON empresas USING GIN (to_tsvector('simple', cnpj));
CREATE INDEX idx_empresas_nome_fantasia ON empresas USING GIN (to_tsvector('simple', nome_fantasia));
CREATE INDEX idx_empresas_razao_social ON empresas USING GIN (to_tsvector('simple', razao_social));
CREATE INDEX idx_empresas_email ON empresas USING GIN (to_tsvector('simple', email));
CREATE INDEX idx_empresas_telefone ON empresas USING GIN (to_tsvector('simple', telefone));
CREATE INDEX idx_empresas_deleted ON empresas(deleted);

-- +goose Down
DROP TABLE IF EXISTS empresas CASCADE;