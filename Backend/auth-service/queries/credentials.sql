-- name: CreateCredential :exec
INSERT INTO passkey_credentials (user_id, credential_id, credential_json)
VALUES ($1, $2, $3);

-- name: GetCredentialJSONByUserID :many
SELECT credential_json
FROM passkey_credentials
WHERE user_id = $1;

-- name: UpdateCredentialJSON :execrows
UPDATE passkey_credentials
SET credential_json = $1
WHERE credential_id = $2;
