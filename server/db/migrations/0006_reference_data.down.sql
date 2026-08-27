DELETE FROM sys_parameters WHERE key IN (
  'contact.whatsapp_primary','contact.whatsapp_secondary','contact.email','contact.instagram',
  'payment.bank_name','payment.bank_account_number','payment.bank_account_name','payment.transfer_note',
  'delivery.free_min_days','delivery.free_min_per_day','delivery.fee',
  'policy.change_deadline_hour','policy.reschedule_fee_pax','policy.off_menu_fee_pax',
  'policy.weekly_package_days','policy.monthly_package_days',
  'order.tier_weekly_min_days','order.tier_monthly_min_days','order.flexi_monthly_max_span',
  'order.range_fill_max_days','order.max_lead_days',
  'site.timezone','site.currency','site.hydration_enabled','site.hydration_poll_seconds'
);
