DROP TABLE IF EXISTS menu_components;
DROP TABLE IF EXISTS menu_days;
DROP TABLE IF EXISTS menu_cycles;
-- btree_gist is left installed: it is cheap, and dropping an extension another
-- migration might later depend on is the more dangerous of the two mistakes.
