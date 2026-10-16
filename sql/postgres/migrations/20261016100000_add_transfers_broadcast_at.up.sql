alter table transfers
  add column broadcast_at timestamp with time zone;

drop index if exists transfers_refund_source_uniq_idx;

create unique index transfers_refund_source_uniq_idx
  on transfers using btree (owner_id, blockchain, source_tx_hash)
  where purpose = 'refund' and (status not in ('failed', 'canceled') or broadcast_at is not null);
