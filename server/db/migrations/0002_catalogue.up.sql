-- 0002_catalogue — plans, their rate tables, and the tier-priced products.
--
-- Money is BIGINT whole rupiah throughout (healthy_catering CLAUDE.md §4, §10).
-- Sen is obsolete in retail, so the rupiah IS the minor unit. No NUMERIC, no
-- floating point, anywhere on a money path.
--
-- The CHECK constraints are not decoration. docs/04-pricing-catalogue.md states
-- monthly <= weekly <= daily as an invariant of the published price list, and
-- tests/pricing.test.js asserts it against the captured page. Encoding it here
-- means the database refuses a rate table that would break the front end's
-- pricing engine, rather than discovering it in production.

CREATE TABLE plans (
    id              UUID        PRIMARY KEY,
    slug            TEXT        NOT NULL UNIQUE,
    -- card_key is the front end's data-sub value. The hydration overlay finds
    -- a card by this string, so it must match the captured markup exactly.
    card_key        TEXT        NOT NULL UNIQUE,
    name            TEXT        NOT NULL,
    description     TEXT        NOT NULL DEFAULT '',
    kcal_min        INT,
    kcal_max        INT,
    -- BR-7.6: Healthy Meal and Bulking Extra never deliver on Sunday.
    delivers_sunday BOOLEAN     NOT NULL DEFAULT TRUE,
    sort_order      INT         NOT NULL DEFAULT 0,
    is_active       BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (kcal_min IS NULL OR kcal_max IS NULL OR kcal_max >= kcal_min)
);

-- One rate row per plan: the five tiers the front end's analyze() selects
-- between (BR-3.1 – BR-3.10).
CREATE TABLE plan_rates (
    plan_id               UUID        PRIMARY KEY REFERENCES plans(id) ON DELETE CASCADE,
    daily                 BIGINT      NOT NULL CHECK (daily > 0),
    weekly_per_day        BIGINT      NOT NULL CHECK (weekly_per_day > 0),
    monthly_per_day       BIGINT      NOT NULL CHECK (monthly_per_day > 0),
    flexi_weekly_per_day  BIGINT      NOT NULL CHECK (flexi_weekly_per_day > 0),
    flexi_monthly_per_day BIGINT      NOT NULL CHECK (flexi_monthly_per_day > 0),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT rates_monotonic       CHECK (monthly_per_day <= weekly_per_day
                                        AND weekly_per_day  <= daily),
    -- A Flexi tier is a smaller discount than the tier it shadows, never a
    -- bigger one. docs/04 §1: "Flexi Bulanan is never cheaper than Bulanan."
    CONSTRAINT flexi_never_cheaper   CHECK (flexi_monthly_per_day >= monthly_per_day
                                        AND flexi_weekly_per_day  >= weekly_per_day),
    CONSTRAINT flexi_never_over_list CHECK (flexi_weekly_per_day <= daily)
);

-- BR-5.x — Regular Catering's 1-5 pax group price table. day_total is the price
-- for the WHOLE group for one day, not per person.
CREATE TABLE plan_pax_prices (
    id         UUID   PRIMARY KEY,
    plan_id    UUID   NOT NULL REFERENCES plans(id) ON DELETE CASCADE,
    rice       TEXT   NOT NULL CHECK (rice   IN ('dengan','tanpa')),
    period     TEXT   NOT NULL CHECK (period IN ('weekly','monthly')),
    pax        INT    NOT NULL CHECK (pax BETWEEN 1 AND 5),
    day_total  BIGINT NOT NULL CHECK (day_total > 0),
    UNIQUE (plan_id, rice, period, pax)
);

-- BR-13.x — Catering Kantor. Priced per pax per day across five pax bands,
-- for two grades and two periods.
CREATE TABLE kantor_periods (
    period     TEXT PRIMARY KEY CHECK (period IN ('mingguan','bulanan')),
    days       INT  NOT NULL CHECK (days > 0),
    label      TEXT NOT NULL,
    sort_order INT  NOT NULL DEFAULT 0
);

CREATE TABLE kantor_rates (
    id               UUID   PRIMARY KEY,
    grade            TEXT   NOT NULL CHECK (grade IN ('reguler','healthy')),
    period           TEXT   NOT NULL REFERENCES kantor_periods(period),
    pax_min          INT    NOT NULL CHECK (pax_min > 0),
    pax_max          INT    NOT NULL,
    rate_per_pax_day BIGINT NOT NULL CHECK (rate_per_pax_day > 0),
    sort_order       INT    NOT NULL DEFAULT 0,
    UNIQUE (grade, period, pax_min),
    CHECK (pax_max >= pax_min)
);

-- BR-14.x — Nasi Bento, Nasi Kuning and Paket Acara. All three price the same
-- way (unit price by quantity band), so they share one shape.
CREATE TABLE tier_products (
    id         UUID PRIMARY KEY,
    slug       TEXT NOT NULL UNIQUE,
    card_key   TEXT NOT NULL UNIQUE,   -- data-sub in the captured markup
    name       TEXT NOT NULL,
    unit       TEXT NOT NULL CHECK (unit IN ('box','pax')),
    min_qty    INT  NOT NULL CHECK (min_qty > 0),
    sort_order INT  NOT NULL DEFAULT 0,
    is_active  BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE tier_packages (
    id         UUID   PRIMARY KEY,
    product_id UUID   NOT NULL REFERENCES tier_products(id) ON DELETE CASCADE,
    name       TEXT   NOT NULL,
    includes   TEXT[] NOT NULL DEFAULT '{}',
    sort_order INT    NOT NULL DEFAULT 0,
    UNIQUE (product_id, name)
);

CREATE TABLE tier_prices (
    id         UUID   PRIMARY KEY,
    package_id UUID   NOT NULL REFERENCES tier_packages(id) ON DELETE CASCADE,
    qty_min    INT    NOT NULL CHECK (qty_min > 0),
    qty_max    INT    NOT NULL,
    label      TEXT   NOT NULL DEFAULT '',
    price      BIGINT NOT NULL CHECK (price > 0),
    sort_order INT    NOT NULL DEFAULT 0,
    UNIQUE (package_id, qty_min),
    CHECK (qty_max >= qty_min)
);

-- BR-6.x — Add-ons. restrict_days holds the allowed weekdays as digit
-- characters, 0=Sunday..6=Saturday, exactly as data-restrict-days does in the
-- captured markup. Empty string means "any day".
CREATE TABLE addons (
    id            UUID   PRIMARY KEY,
    scope         TEXT   NOT NULL CHECK (scope IN ('daily','bento','kantor')),
    code          TEXT   NOT NULL,       -- the checkbox `value`
    label         TEXT   NOT NULL,       -- the visible chip text
    price         BIGINT NOT NULL CHECK (price >= 0),
    restrict_days TEXT   NOT NULL DEFAULT '' CHECK (restrict_days ~ '^[0-6]*$'),
    -- BR-6.8: the Flexi meat cap. 5 means "at most one portion per 5 pax on any
    -- Flexi tier". NULL means the add-on always charges per full pax.
    flexi_portion_per_pax INT CHECK (flexi_portion_per_pax IS NULL OR flexi_portion_per_pax > 0),
    sort_order    INT     NOT NULL DEFAULT 0,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    UNIQUE (scope, code)
);

CREATE INDEX ON plan_pax_prices (plan_id);
CREATE INDEX ON kantor_rates    (grade, period);
CREATE INDEX ON tier_packages   (product_id);
CREATE INDEX ON tier_prices     (package_id);
CREATE INDEX ON addons          (scope) WHERE is_active;
