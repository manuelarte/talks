-- name: GetAccounts :many
SELECT * FROM accounts
    LIMIT @limit
    OFFSET @offset;

-- name: GetAccount :one
SELECT * FROM accounts
    WHERE id = @id;

-- name: CountAccounts :one
SELECT count(*) FROM accounts;

-- name: SaveAmount :exec
UPDATE accounts SET amount = @amount WHERE id = @id
