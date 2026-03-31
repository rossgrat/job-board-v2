-- name: GetActiveFilterGroups :many
SELECT * FROM filter_group WHERE is_active = true;

-- name: GetFilterConditionsByGroupID :many
SELECT * FROM filter_condition WHERE filter_group_id = $1;
