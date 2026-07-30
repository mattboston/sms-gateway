-- +goose Up

-- Paginated listings query "WHERE direction = ? [AND status = ?] ORDER BY
-- created_at DESC, id DESC". The existing indexes cover either the filter or
-- the ordering but never both, so every page forced a sort over all matching
-- rows. These composite indexes let SQLite satisfy the filter and the ordering
-- from one index and stop as soon as LIMIT is met.
--
-- id is included as the tiebreaker because created_at only has second
-- granularity and is therefore not unique.
CREATE INDEX idx_messages_direction_created_at ON messages(direction, created_at DESC, id DESC);
CREATE INDEX idx_messages_direction_status_created_at ON messages(direction, status, created_at DESC, id DESC);

-- +goose Down

DROP INDEX IF EXISTS idx_messages_direction_status_created_at;
DROP INDEX IF EXISTS idx_messages_direction_created_at;
