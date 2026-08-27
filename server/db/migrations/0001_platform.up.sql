-- 0001_platform — the two tables everything else leans on.
--
-- sys_parameters is the house convention (healthy_catering CLAUDE.md §7):
-- anything that could change without a code change is a row here, not a
-- constant. content_revision is what lets the hydration overlay ask "has
-- anything changed?" in one cheap query instead of diffing a 60 KB document.

CREATE TABLE sys_parameters (
    key         TEXT PRIMARY KEY,
    value       TEXT        NOT NULL,
    value_type  TEXT        NOT NULL CHECK (value_type IN ('string','int','bool','json')),
    label       TEXT        NOT NULL,
    description TEXT        NOT NULL DEFAULT '',
    group_name  TEXT        NOT NULL DEFAULT 'general',
    sort_order  INT         NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE sys_parameters IS
    'Configurable values with full CRUD behind an admin permission. Never a code constant.';

-- Exactly one row, forever. The BOOLEAN-primary-key-CHECK-TRUE trick is the
-- standard Postgres singleton: a second row cannot be inserted.
CREATE TABLE content_revision (
    only_row   BOOLEAN     PRIMARY KEY DEFAULT TRUE CHECK (only_row),
    revision   BIGINT      NOT NULL DEFAULT 1,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO content_revision (only_row) VALUES (TRUE);

-- Bumped by the API whenever any content table is written. The overlay sends
-- If-None-Match with the revision; an unchanged revision answers 304 and the
-- 60 KB payload never crosses the wire.
CREATE FUNCTION bump_content_revision() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE content_revision
       SET revision = revision + 1, updated_at = now()
     WHERE only_row;
    RETURN NULL;
END;
$$;
