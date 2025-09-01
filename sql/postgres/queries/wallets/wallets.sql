-- name: MaxSequence :one
with maxProcessing as (
	select coalesce(max(sequence),-1)::int as sequence from processing_wallets pw where pw.blockchain = $1 and pw.owner_id = $2
),
maxHot as (
	select coalesce(max(sequence),-1)::int as sequence from hot_wallets hw where hw.blockchain = $1 and hw.owner_id = $2
)
select
  coalesce(
	case
		when mp.sequence > mh.sequence then mp.sequence else mh.sequence
	end, 0
  )::int
from maxProcessing mp, maxHot mh;
