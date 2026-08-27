-- 0003_menu — the weekly menu rotation.
--
-- This is the table that exists because of BR-15.1: in the captured page the
-- weekly menus are hard-coded markup, so publishing next week's menu means
-- editing HTML. docs/09-open-questions.md raises it as Q-12. Here the menu is
-- data on a cycle, which is what that question asked for.
--
-- A cycle is one Monday-anchored week. Each (cycle, plan, date) is one menu
-- day, and a menu day is an ordered list of components with gram weights --
-- the same shape the page renders as
-- "Nasi Merah (100g), Ayam Goreng Serundeng (90g), ... ±455 kkal".

CREATE TABLE menu_cycles (
    id           UUID        PRIMARY KEY,
    iso_year     INT         NOT NULL CHECK (iso_year BETWEEN 2020 AND 2100),
    iso_week     INT         NOT NULL CHECK (iso_week BETWEEN 1 AND 53),
    starts_on    DATE        NOT NULL,
    ends_on      DATE        NOT NULL,
    -- The exact string the page shows, e.g. "Minggu ke-35 · 24–28 Agustus 2026".
    -- Kept as text rather than derived: the business writes these by hand and
    -- the em-dash/spacing is part of the house style.
    label        TEXT        NOT NULL,
    is_published BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (iso_year, iso_week),
    CHECK (ends_on >= starts_on),
    -- A cycle is a week. Anything wider is a data-entry mistake that would
    -- silently make two cycles claim the same day.
    CHECK (ends_on - starts_on <= 6)
);

-- Two published cycles must never overlap, or "which menu applies on this
-- date" has no single answer. An exclusion constraint is the only way to say
-- that in the schema rather than in application code.
CREATE EXTENSION IF NOT EXISTS btree_gist;
ALTER TABLE menu_cycles ADD CONSTRAINT menu_cycles_no_overlap
    EXCLUDE USING gist (
        daterange(starts_on, ends_on, '[]') WITH &&
    ) WHERE (is_published);

CREATE TABLE menu_days (
    id          UUID    PRIMARY KEY,
    cycle_id    UUID    NOT NULL REFERENCES menu_cycles(id) ON DELETE CASCADE,
    plan_id     UUID    NOT NULL REFERENCES plans(id)       ON DELETE CASCADE,
    serve_date  DATE    NOT NULL,
    -- BR-15.3: Thursday is the meat day, marked with a star on the page.
    is_meat_day BOOLEAN NOT NULL DEFAULT FALSE,
    kcal        INT     CHECK (kcal IS NULL OR kcal > 0),
    UNIQUE (cycle_id, plan_id, serve_date)
);

CREATE TABLE menu_components (
    id          UUID PRIMARY KEY,
    menu_day_id UUID NOT NULL REFERENCES menu_days(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    grams       INT  CHECK (grams IS NULL OR grams > 0),
    sort_order  INT  NOT NULL DEFAULT 0
);

CREATE INDEX ON menu_days       (cycle_id, plan_id);
CREATE INDEX ON menu_days       (serve_date);
CREATE INDEX ON menu_components (menu_day_id, sort_order);

-- A menu change is the single most common content edit, so every menu table
-- bumps the revision the overlay polls.
CREATE TRIGGER menu_cycles_bump     AFTER INSERT OR UPDATE OR DELETE ON menu_cycles
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER menu_days_bump       AFTER INSERT OR UPDATE OR DELETE ON menu_days
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER menu_components_bump AFTER INSERT OR UPDATE OR DELETE ON menu_components
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
