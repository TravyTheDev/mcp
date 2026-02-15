-- +goose Up
-- +goose StatementBegin
CREATE TABLE humans (
    id integer primary key autoincrement,
    first_name varchar(255),
    last_name varchar(255),
    date_of_birth TEXT,
    has_allergies tinyint,
    bio TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE humans;
-- +goose StatementEnd

