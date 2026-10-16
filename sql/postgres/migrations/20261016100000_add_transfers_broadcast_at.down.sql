drop index if exists transfers_refund_source_uniq_idx;

create unique index transfers_refund_source_uniq_idx
  on transfers using btree (owner_id, blockchain, source_tx_hash)
  where purpose = 'refund' and status not in ('failed', 'canceled');

alter table transfers
  drop column if exists broadcast_at;
