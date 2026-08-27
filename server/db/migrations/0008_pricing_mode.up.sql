-- 0008_pricing_mode — how much authority the server's calculator has.
--
-- verify        the overlay shadow-quotes and reports disagreement, and the
--               page's own figure is what gets charged. Zero risk.
-- authoritative the server's figure is what goes in the cart.
--
-- Defaults to verify. The honest order is to run verify until the parity log is
-- quiet on real customer input, and only then hand over authority — verify is
-- what tells you the two engines agree on YOUR data rather than on a fixture.
INSERT INTO sys_parameters (key, value, value_type, label, description, group_name, sort_order)
VALUES ('order.pricing_mode', 'verify', 'string', 'Mode kalkulasi harga',
        'verify = server hanya membandingkan dan melaporkan selisih; authoritative = angka server yang dipakai di keranjang.',
        'order', 5);
