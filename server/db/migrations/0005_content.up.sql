-- 0005_content — the marketing copy that changes without a code change.
--
-- Stats, testimonials, pillars, the timeline, contact routes and the bank
-- account. One table rather than six: they are all "an ordered list of small
-- labelled things belonging to a slot on a page", and six near-identical tables
-- would buy nothing. `meta` carries the handful of per-kind extras (a stat's
-- numeric target, a timeline entry's year) as JSONB.

CREATE TABLE content_blocks (
    id         UUID        PRIMARY KEY,
    kind       TEXT        NOT NULL CHECK (kind IN
                   ('stat','testimonial','pillar','timeline','contact','value','area_card','faq')),
    slot       TEXT        NOT NULL,   -- page/section, e.g. 'home.stats'
    heading    TEXT        NOT NULL DEFAULT '',
    body       TEXT        NOT NULL DEFAULT '',
    meta       JSONB       NOT NULL DEFAULT '{}',
    sort_order INT         NOT NULL DEFAULT 0,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON content_blocks (slot, sort_order) WHERE is_active;
CREATE INDEX ON content_blocks (kind);

CREATE TRIGGER content_blocks_bump AFTER INSERT OR UPDATE OR DELETE ON content_blocks
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();

-- The catalogue tables bump the revision too. They are declared here rather
-- than in 0002 because bump_content_revision() and the tables must both exist,
-- and keeping every trigger declaration in one place makes them auditable.
CREATE TRIGGER plans_bump           AFTER INSERT OR UPDATE OR DELETE ON plans
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER plan_rates_bump      AFTER INSERT OR UPDATE OR DELETE ON plan_rates
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER plan_pax_prices_bump AFTER INSERT OR UPDATE OR DELETE ON plan_pax_prices
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER kantor_rates_bump    AFTER INSERT OR UPDATE OR DELETE ON kantor_rates
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER tier_products_bump   AFTER INSERT OR UPDATE OR DELETE ON tier_products
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER tier_packages_bump   AFTER INSERT OR UPDATE OR DELETE ON tier_packages
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER tier_prices_bump     AFTER INSERT OR UPDATE OR DELETE ON tier_prices
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER addons_bump          AFTER INSERT OR UPDATE OR DELETE ON addons
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
CREATE TRIGGER sys_parameters_bump  AFTER INSERT OR UPDATE OR DELETE ON sys_parameters
    FOR EACH STATEMENT EXECUTE FUNCTION bump_content_revision();
