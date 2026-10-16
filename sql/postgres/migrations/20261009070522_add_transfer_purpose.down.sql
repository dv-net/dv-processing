drop index if exists transfers_execute_after_idx;
drop index if exists transfers_refund_source_uniq_idx;

alter table transfers
  drop constraint if exists transfers_hot_external_purpose_check,
  drop constraint if exists transfers_hot_external_check,
  drop constraint if exists transfers_withdrawal_2fa_check,
  drop constraint if exists transfers_external_amount_check,
  drop constraint if exists transfers_refund_source_check,
  drop constraint if exists transfers_wallet_to_type_check,
  drop constraint if exists transfers_purpose_check;

alter table transfers
  drop column if exists two_factor_confirmed_at,
  drop column if exists execute_after,
  drop column if exists wallet_to_type,
  drop column if exists source_tx_hash,
  drop column if exists purpose;
