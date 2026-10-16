alter table transfers
  add column purpose                 varchar(30) not null default 'default',
  add column source_tx_hash          varchar(100),
  add column wallet_to_type          varchar(30),
  add column execute_after           timestamp with time zone,
  add column two_factor_confirmed_at timestamp with time zone;

alter table transfers
  add constraint transfers_purpose_check
    check (purpose in ('default', 'withdrawal', 'refund')),

  add constraint transfers_wallet_to_type_check
    check (wallet_to_type is null or wallet_to_type in ('hot', 'cold', 'processing', 'external')),

  -- source_tx_hash is the refunded deposit and is set for refunds only
  add constraint transfers_refund_source_check
    check ((purpose = 'refund') = (source_tx_hash is not null and source_tx_hash != '')),

  -- transfers to external addresses must have an exact amount
  add constraint transfers_external_amount_check
    check (wallet_to_type is distinct from 'external' or (amount is not null and not whole_amount)),

  -- withdrawals to external addresses are allowed only with 2FA
  add constraint transfers_withdrawal_2fa_check
    check (purpose != 'withdrawal' or two_factor_confirmed_at is not null),

  -- hot wallet -> external address: only a withdrawal or a refund,
  -- and either confirmed by 2FA or a refund delayed by the cooldown
  add constraint transfers_hot_external_check
    check (
      wallet_from_type != 'hot'
      or wallet_to_type is distinct from 'external'
      or two_factor_confirmed_at is not null
      or (purpose = 'refund' and execute_after is not null)
    ),
  add constraint transfers_hot_external_purpose_check
    check (
      wallet_from_type != 'hot'
      or wallet_to_type is distinct from 'external'
      or purpose in ('withdrawal', 'refund')
    );

-- one active refund per deposit
create unique index transfers_refund_source_uniq_idx
  on transfers using btree (owner_id, blockchain, source_tx_hash)
  where purpose = 'refund' and status not in ('failed', 'canceled');

-- scheduled transfers waiting for execution
create index transfers_execute_after_idx
  on transfers using btree (execute_after)
  where status = 'new' and execute_after is not null;
