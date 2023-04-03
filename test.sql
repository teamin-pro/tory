-- name: test-comments
SELECT 1 +
       -- comment
       2;

-- name: test-comments-after-semicolon
SELECT 1 +
       2; -- comment

-- name: test-arguments
SELECT name FROM users WHERE id = :id AND name ILIKE :q;

-- name: test.name-with-dots
SELECT 1;
