-- 0007_pricing_rules — the thresholds the tier classifier compares against.
--
-- Until now these lived as literals inside the front end's analyze():
-- "n >= 20", "n <= 14", "span <= 45", "span <= 31". Changing the definition of
-- Paket Bulanan meant editing JavaScript inside a 6.7 MB frozen capture.
--
-- One singleton row with typed columns rather than key/value pairs, because
-- these values are only meaningful RELATIVE TO EACH OTHER: a monthly minimum
-- below the weekly minimum makes the Bulanan branch unreachable, and a Flexi
-- ceiling above the monthly floor makes the full-price branch unreachable.
-- Typed columns let the database refuse those combinations outright; key/value
-- rows could not express a single one of these CHECKs.
--
-- Defaults reproduce the captured page exactly (BR-3.1 – BR-3.13), so seeding
-- changes nothing about what the site charges.

CREATE TABLE pricing_rules (
    only_row BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (only_row),

    -- BR-3.4 — consecutive dates from here up qualify for Mingguan.
    weekly_min_days                   INT NOT NULL DEFAULT 5,

    -- BR-3.1 — consecutive dates from here up qualify for Bulanan.
    monthly_min_days                  INT NOT NULL DEFAULT 20,

    -- BR-3.2 / BR-3.3 — a consecutive run that crosses a calendar-week boundary
    -- gets Flexi Mingguan only up to this length; beyond it, and short of
    -- monthly_min_days, the order pays the full daily rate.
    consecutive_flexi_weekly_max_days INT NOT NULL DEFAULT 14,

    -- BR-3.8 — scattered dates totalling monthly_min_days still get the Flexi
    -- Bulanan discount if they fall inside this window.
    flexi_monthly_max_span_days       INT NOT NULL DEFAULT 45,

    -- BR-3.11 — a clean Mon-Fri / Mon-Sat routine counts as full Bulanan when
    -- it fits inside this window.
    weekday_routine_max_span_days     INT NOT NULL DEFAULT 31,

    -- BR-3.12 — an order that "nyambung" across two weeks counts as Mingguan
    -- when it fits inside this window...
    weekly_routine_max_span_days      INT NOT NULL DEFAULT 14,
    -- ...and at least one calendar week inside it holds this many dates.
    weekly_routine_min_days_in_week   INT NOT NULL DEFAULT 5,

    -- BR-5.4 — above this pax count the group price table stops deepening and
    -- extends linearly from its last row.
    pax_table_max_pax                 INT NOT NULL DEFAULT 5,

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Each of these rules out a combination that would silently make one branch
    -- of the classifier unreachable.
    CONSTRAINT weekly_min_sane      CHECK (weekly_min_days >= 1),
    CONSTRAINT monthly_above_weekly CHECK (monthly_min_days > weekly_min_days),
    CONSTRAINT flexi_ceiling_sane   CHECK (consecutive_flexi_weekly_max_days >= weekly_min_days
                                       AND consecutive_flexi_weekly_max_days < monthly_min_days),
    CONSTRAINT flexi_span_sane      CHECK (flexi_monthly_max_span_days >= monthly_min_days),
    CONSTRAINT weekday_span_sane    CHECK (weekday_routine_max_span_days >= monthly_min_days),
    CONSTRAINT weekly_span_sane     CHECK (weekly_routine_max_span_days >= weekly_min_days),
    CONSTRAINT week_count_sane      CHECK (weekly_routine_min_days_in_week BETWEEN 1 AND 7),
    CONSTRAINT pax_table_sane       CHECK (pax_table_max_pax >= 1)
);

INSERT INTO pricing_rules (only_row) VALUES (TRUE);

CREATE TRIGGER pricing_rules_bump AFTER INSERT OR UPDATE OR DELETE ON pricing_rules
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
