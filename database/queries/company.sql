-- name: CreateCompany :one
INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetActiveCompanies :many
SELECT * FROM company WHERE is_active = true;

-- name: GetCompanyByName :one
SELECT * FROM company WHERE name = $1;

-- name: GetCompanyByID :one
SELECT * FROM company WHERE id = $1;

-- name: SetCompanyActive :exec
UPDATE company SET is_active = $2 WHERE id = $1;
