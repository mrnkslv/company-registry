-- +goose Up
-- +goose StatementBegin
CREATE TYPE company_type AS ENUM ('Corporations', 'NonProfit', 'Cooperative', 'Sole Proprietorship');

CREATE TABLE companies (
                         id UUID PRIMARY KEY NOT NULL ,
                         name TEXT NOT NULL CHECK (LENGTH(name) <= 15),
                         description TEXT CHECK (LENGTH(description) <= 3000),
                         amount_of_employees INT NOT NULL,
                         registered BOOLEAN NOT NULL,
                         type company_type NOT NULL
    );

CREATE UNIQUE INDEX idx_companies_name ON companies (name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE companies;
DROP TYPE company_type;
-- +goose StatementEnd
