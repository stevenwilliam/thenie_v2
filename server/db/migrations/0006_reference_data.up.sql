-- 0006_reference_data — the sys_parameters rows.
--
-- Every value here is one that appears in the captured page as a hard-coded
-- constant and could change without a code change (house convention §7). The
-- BR-x.y reference in each description points at docs/02-business-rules.md, so
-- an admin editing a value can find out what it governs.
--
-- Several of these are fees the page STATES but never CHARGES (BR-9.7,
-- BR-10.4, Q-5). They are recorded here so the rebuild has the real numbers in
-- one place; whether the calculator applies them is a product decision, not a
-- schema one.

INSERT INTO sys_parameters (key, value, value_type, label, description, group_name, sort_order) VALUES
  ('contact.whatsapp_primary',     '62818100523',            'string', 'WhatsApp utama',
   'Nomor tujuan semua order dan CTA. BR-12.9.',                                   'contact',  10),
  ('contact.whatsapp_secondary',   '62817771123',            'string', 'WhatsApp kedua',
   'Hanya muncul di halaman Kontak. Perannya belum jelas - Q-18.',                 'contact',  20),
  ('contact.email',                'thenie.resto@gmail.com', 'string', 'Email',
   'Ditampilkan di Kontak dan footer.',                                            'contact',  30),
  ('contact.instagram',            'thenie.id',              'string', 'Instagram handle',
   'Tanpa tanda @.',                                                               'contact',  40),

  ('payment.bank_name',            'BCA',                    'string', 'Nama bank',
   'BR-8.1: transfer bank adalah satu-satunya metode pembayaran.',                 'payment',  10),
  ('payment.bank_account_number',  '8660281402',             'string', 'Nomor rekening',
   'BR-8.2. Ditampilkan dengan format 8660-281-402.',                              'payment',  20),
  ('payment.bank_account_name',    'R Bg Andreas Kurnianto', 'string', 'Nama pemilik rekening',
   'BR-8.2: nama pribadi, bukan badan usaha - Q-17.',                              'payment',  30),
  ('payment.transfer_note',        'Catering atas nama ...', 'string', 'Berita transfer',
   'BR-8.3: wajib dicantumkan agar mudah dicek.',                                  'payment',  40),

  ('delivery.free_min_days',       '5',                      'int',    'Minimum hari gratis ongkir',
   'BR-10.3. Tidak dihitung kalkulator - Q-5.',                                    'delivery', 10),
  ('delivery.free_min_per_day',    '26000',                  'int',    'Minimum nilai menu/hari (Rp)',
   'BR-10.3. Di bawah ini kena ongkir.',                                           'delivery', 20),
  ('delivery.fee',                 '5000',                   'int',    'Ongkir per pengiriman (Rp)',
   'BR-10.3. Dinyatakan di halaman tapi tidak pernah ditagih - BR-10.4.',          'delivery', 30),

  ('policy.change_deadline_hour',  '17',                     'int',    'Batas perubahan H-1 (jam WIB)',
   'BR-9.1: semua perubahan maksimal H-1 pukul 17.00 WIB.',                        'policy',   10),
  ('policy.reschedule_fee_pax',    '10000',                  'int',    'Biaya pindah minggu (Rp/pax)',
   'BR-9.4. Tidak dihitung kalkulator - Q-5.',                                     'policy',   20),
  ('policy.off_menu_fee_pax',      '5000',                   'int',    'Biaya request di luar menu (Rp/pax)',
   'BR-9.6. Tidak dihitung kalkulator - Q-5.',                                     'policy',   30),
  ('policy.weekly_package_days',   '7',                      'int',    'Masa berlaku paket mingguan (hari)',
   'BR-9.5.',                                                                      'policy',   40),
  ('policy.monthly_package_days',  '30',                     'int',    'Masa berlaku paket bulanan (hari)',
   'BR-9.5: 30 hari sejak tanggal pembayaran.',                                    'policy',   50),

  ('order.tier_weekly_min_days',   '5',                      'int',    'Minimum hari paket Mingguan',
   'BR-3.4.',                                                                      'order',    10),
  ('order.tier_monthly_min_days',  '20',                     'int',    'Minimum hari paket Bulanan',
   'BR-3.1.',                                                                      'order',    20),
  ('order.flexi_monthly_max_span', '45',                     'int',    'Rentang maks Flexi Bulanan (hari)',
   'BR-3.8.',                                                                      'order',    30),
  ('order.range_fill_max_days',    '60',                     'int',    'Batas Isi Rentang (hari)',
   'BR-7.13.',                                                                     'order',    40),
  ('order.max_lead_days',          '0',                      'int',    'Batas pesan ke depan (hari, 0 = tanpa batas)',
   'BR-7.8: saat ini tanpa batas - Q-10 mengusulkan 90.',                          'order',    50),

  ('site.timezone',                'Asia/Jakarta',           'string', 'Zona waktu operasional',
   'Semua cut-off dihitung di zona ini, bukan zona server.',                       'site',     10),
  ('site.currency',                'IDR',                    'string', 'Mata uang',
   'Disimpan sebagai BIGINT rupiah penuh.',                                        'site',     20),
  ('site.hydration_enabled',       'true',                   'bool',   'Aktifkan hydration overlay',
   'Kill switch. false = halaman memakai konten bawaan capture saja.',             'site',     30),
  ('site.hydration_poll_seconds',  '0',                      'int',    'Interval cek ulang konten (detik, 0 = sekali saat load)',
   'Overlay memeriksa revisi konten; 0 berarti tidak polling.',                    'site',     40);
