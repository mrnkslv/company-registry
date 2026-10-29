-- +goose Up
SELECT 'up SQL query';

CREATE TABLE users 
(
    id serial not null unique,
    name varchar (255) not null,
    username varchar(255) not null unique,
    password_hash varchar(255) not null 
);


insert into users(name, username, password_hash)
values ('testname', 'testusername', '6a666768646c6b6f76696f6a6b66646b6fe38ad214943daad1d64c102faec29de4afe9da3d');

-- +goose Down
SELECT 'down SQL query';

DROP TABLE users;