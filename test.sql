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

-- name: test-sum
SELECT :x::int + :y::int;

-- name: test-do-block
-- A dollar-quoted DO statement contains semicolons of its own; the
-- query parser must not treat them as the end of the named query.
do $$ begin
    create type test_dollar_color as enum ('red', 'green');
exception when duplicate_object then null; end $$;

-- name: test-do-block-multiple-statements
do $$ begin
    perform 1;
    perform 2;
end $$;
