-- 0004_operations — service areas and delivery windows.
--
-- BR-11.4 / Q-27: the captured page carries TWO area lists that do not match.
-- The order form's dropdown offers nine values; the marketing pages advertise a
-- different five plus three "expansion" areas. Rather than pick a winner and
-- lose the distinction, both facts are modelled: is_orderable drives the
-- dropdown, is_advertised drives the marketing copy. That makes the mismatch
-- visible and editable instead of hidden in two places in the markup.

CREATE TABLE service_areas (
    id            UUID    PRIMARY KEY,
    name          TEXT    NOT NULL UNIQUE,
    is_orderable  BOOLEAN NOT NULL DEFAULT TRUE,   -- appears in the Area dropdown
    is_advertised BOOLEAN NOT NULL DEFAULT TRUE,   -- appears in marketing copy
    -- "Lainnya" is the catch-all option; it is not a real place and must not be
    -- advertised or counted as coverage (BR-11.3).
    is_catch_all  BOOLEAN NOT NULL DEFAULT FALSE,
    sort_order    INT     NOT NULL DEFAULT 0
);

-- Exactly one catch-all, or the dropdown grows two "other" entries.
CREATE UNIQUE INDEX service_areas_one_catch_all ON service_areas ((TRUE)) WHERE is_catch_all;

-- BR-10.1 / BR-7.4 — the five delivery windows and their same-day cut-offs.
--
-- same_day_cutoff_hour is the hour (Asia/Jakarta) after which this window can no
-- longer be chosen for TODAY. NULL means the window is never available same-day,
-- which is what the two Pagi windows do. 24 means it is always available, which
-- is what "Request (dikonfirmasi admin)" does -- the admin confirms by hand.
CREATE TABLE delivery_windows (
    id                   UUID    PRIMARY KEY,
    code                 TEXT    NOT NULL UNIQUE,
    label                TEXT    NOT NULL,   -- chip text, e.g. "Pagi 06.00–07.00"
    value                TEXT    NOT NULL,   -- the radio `value`, written into the WA message
    note                 TEXT    NOT NULL DEFAULT '',
    is_default           BOOLEAN NOT NULL DEFAULT FALSE,
    same_day_cutoff_hour INT     CHECK (same_day_cutoff_hour IS NULL
                                     OR same_day_cutoff_hour BETWEEN 0 AND 24),
    sort_order           INT     NOT NULL DEFAULT 0
);

-- BR-12.7 resets every card to the default window, so there has to be exactly
-- one. A partial unique index on a constant is the standard way to say
-- "at most one row satisfying this predicate".
CREATE UNIQUE INDEX delivery_windows_one_default ON delivery_windows ((TRUE)) WHERE is_default;

CREATE TRIGGER service_areas_bump    AFTER INSERT OR UPDATE OR DELETE ON service_areas
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER delivery_windows_bump AFTER INSERT OR UPDATE OR DELETE ON delivery_windows
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
