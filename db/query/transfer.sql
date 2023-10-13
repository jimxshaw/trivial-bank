-- name: CreateTransfer :one
INSERT INTO transfers (
  from_account_id,
  to_account_id,
  amount
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetTransfer :one
SELECT * 
FROM transfers
WHERE id = $1 LIMIT 1;

-- name: ListTransfers :many
SELECT t.id, t.from_account_id, t.to_account_id, t.amount, t.created_at 
FROM transfers AS t
JOIN accounts AS a1 ON t.from_account_id = a1.id
JOIN accounts AS a2 ON t.to_account_id = a2.id
WHERE (a1.user_id = $1 OR a2.user_id = $1)
LIMIT $2 
OFFSET $3;
