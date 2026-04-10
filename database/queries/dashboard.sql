-- name: ListFilteredJobs :many
SELECT
    cj.id,
    cj.title,
    cj.salary_min,
    cj.salary_max,
    cj.level,
    cj.category,
    cj.relevance,
    rj.id AS raw_job_id,
    rj.url,
    rj.discovered_at,
    rj.user_status,
    c.name AS company_name,
    c.favicon_url AS company_favicon_url,
    COALESCE(
        array_agg(DISTINCT cjl.setting || ':' || cjl.country || ':' || COALESCE(cjl.city, ''))
        FILTER (WHERE cjl.id IS NOT NULL),
        '{}'
    )::text[] AS locations,
    COALESCE(
        array_agg(DISTINCT cjt.name)
        FILTER (WHERE cjt.id IS NOT NULL),
        '{}'
    )::text[] AS technologies,
    EXISTS(SELECT 1 FROM eval_entry ee WHERE ee.raw_job_id = rj.id) AS has_eval
FROM classified_job cj
JOIN raw_job rj ON rj.id = cj.raw_job_id
JOIN company c ON c.id = rj.company_id
LEFT JOIN classified_job_location cjl ON cjl.classified_job_id = cj.id
LEFT JOIN classified_job_technology cjt ON cjt.classified_job_id = cj.id
WHERE cj.is_current = true
  AND rj.deleted_at IS NULL
  AND cj.status IN ('accepted', 'filtered_relevance')
  AND (@relevance::text = '' OR cj.relevance = @relevance)
  AND (@user_status::text = '' OR
       (@user_status = 'new' AND rj.user_status IS NULL) OR
       (@user_status != 'new' AND rj.user_status = @user_status))
  AND (@company_name::text = '' OR c.name = @company_name)
GROUP BY cj.id, rj.id, c.id
ORDER BY
    CASE WHEN rj.user_status IS NULL THEN 0
         WHEN rj.user_status = 'applied' THEN 1
         WHEN rj.user_status = 'tabled' THEN 2
         ELSE 3 END,
    CASE WHEN cj.relevance = 'strong_match' THEN 0 ELSE 1 END,
    rj.discovered_at DESC;

-- name: ListDashboardJobs :many
SELECT
    cj.id,
    cj.title,
    cj.salary_min,
    cj.salary_max,
    cj.level,
    cj.category,
    cj.relevance,
    rj.id AS raw_job_id,
    rj.url,
    rj.discovered_at,
    rj.user_status,
    c.name AS company_name,
    c.favicon_url AS company_favicon_url,
    COALESCE(
        array_agg(DISTINCT cjl.setting || ':' || cjl.country || ':' || COALESCE(cjl.city, ''))
        FILTER (WHERE cjl.id IS NOT NULL),
        '{}'
    )::text[] AS locations,
    COALESCE(
        array_agg(DISTINCT cjt.name)
        FILTER (WHERE cjt.id IS NOT NULL),
        '{}'
    )::text[] AS technologies,
    EXISTS(SELECT 1 FROM eval_entry ee WHERE ee.raw_job_id = rj.id) AS has_eval
FROM classified_job cj
JOIN raw_job rj ON rj.id = cj.raw_job_id
JOIN company c ON c.id = rj.company_id
LEFT JOIN classified_job_location cjl ON cjl.classified_job_id = cj.id
LEFT JOIN classified_job_technology cjt ON cjt.classified_job_id = cj.id
WHERE cj.is_current = true
  AND rj.deleted_at IS NULL
  AND cj.status = 'accepted'
  AND cj.relevance IN ('strong_match', 'good_match')
  AND (rj.user_status IS NULL OR rj.user_status = 'tabled')
GROUP BY cj.id, rj.id, c.id
ORDER BY
    CASE WHEN rj.user_status IS NULL THEN 0
         WHEN rj.user_status = 'tabled' THEN 1
         ELSE 2 END,
    CASE WHEN cj.relevance = 'strong_match' THEN 0 ELSE 1 END,
    rj.discovered_at DESC;
