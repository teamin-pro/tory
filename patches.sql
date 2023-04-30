-- name: tory.create-table-db-version
CREATE TABLE IF NOT EXISTS db_version (
    version INTEGER NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- name: tory.upsert-db-version
WITH
    current_version AS (
        SELECT version, created_at FROM db_version
        ORDER BY version DESC LIMIT 1
    ),
    initial_version AS (
        INSERT INTO db_version (version)
        SELECT (:version) WHERE NOT EXISTS (SELECT * FROM current_version)
        RETURNING version, created_at
    )
SELECT * FROM current_version
UNION ALL
SELECT * FROM initial_version;

-- name: tory.update-db-version
INSERT INTO db_version (version) VALUES (:version)
ON CONFLICT (version) DO NOTHING;
