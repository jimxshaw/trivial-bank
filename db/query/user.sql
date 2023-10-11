-- name: CreateUser :one
INSERT INTO users (
  first_name,
  last_name,
  email,
  username,
  password
) VALUES (
  $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUser :one
SELECT * 
FROM users
WHERE username = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * 
FROM users
WHERE id = $1 LIMIT 1;
