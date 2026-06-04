-- name: CreateCompany :one
INSERT INTO company (id, name, fetch_type, fetch_config, favicon_url)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListCompanies :many
SELECT * FROM company ORDER BY name;

-- name: ListCompaniesWithJobCounts :many
SELECT
    c.id,
    c.name,
    c.fetch_type,
    c.favicon_url,
    c.is_active,
    COUNT(cj.id) AS total,
    COUNT(cj.id) FILTER (WHERE cj.status != 'non_technical') AS technical,
    COUNT(cj.id) FILTER (WHERE cj.status = 'accepted') AS accepted,
    COUNT(cj.id) FILTER (WHERE cj.status = 'filtered_relevance') AS filtered_relevance,
    COUNT(cj.id) FILTER (WHERE cj.status = 'filtered_level') AS filtered_level,
    COUNT(cj.id) FILTER (WHERE cj.status = 'filtered_location') AS filtered_location,
    COUNT(cj.id) FILTER (WHERE cj.status = 'non_technical') AS non_technical,
    COUNT(cj.id) FILTER (WHERE cj.status = 'pending') AS pending,
    COUNT(cj.id) FILTER (WHERE cj.status = 'dead') AS dead
FROM company c
LEFT JOIN raw_job rj ON rj.company_id = c.id AND rj.deleted_at IS NULL
LEFT JOIN classified_job cj ON cj.raw_job_id = rj.id AND cj.is_current = true
GROUP BY c.id
ORDER BY c.name;

-- name: GetActiveCompanies :many
SELECT * FROM company WHERE is_active = true;

-- name: GetCompanyByName :one
SELECT * FROM company WHERE name = $1;

-- name: GetCompanyByID :one
SELECT * FROM company WHERE id = $1;

-- name: SetCompanyActive :exec
UPDATE company SET is_active = $2 WHERE id = $1;
