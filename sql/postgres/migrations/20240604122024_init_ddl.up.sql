-- Clients
create table clients
(
  id           uuid not null primary key default gen_random_uuid(),
  secret_key  varchar(255) not null check (secret_key != ''),
  callback_url varchar(2048) not null check (callback_url != ''),
  created_at   timestamp with time zone not null default (timezone('utc', now())),
  updated_at   timestamp with time zone
);

-- Owners
create table owners
(
  id          uuid not null primary key default gen_random_uuid(),
  external_id varchar(255) not null unique,
  client_id   uuid not null constraint fk_owners_cid references clients,
  mnemonic    varchar(255) not null check (mnemonic != ''),
  pass_phrase varchar(255),
  otp_secret  varchar(255),
  otp_confirmed bool not null default false,
  created_at  timestamp with time zone not null default (timezone('utc', now())),
  updated_at  timestamp with time zone
);
create index owners_external_id_idx on owners using btree (external_id);

-- Cold wallets
create table cold_wallets
(
  id          uuid not null primary key default gen_random_uuid(),
  blockchain  varchar(255) not null check (blockchain != ''),
  address     varchar(255) not null check (address != ''),
  owner_id    uuid not null constraint fk_wallets_oid references owners,
  is_active   bool not null default true,
  is_dirty    bool not null default false,
  created_at  timestamp with time zone not null default (timezone('utc', now())),
  updated_at  timestamp with time zone
);
create unique index uni_idx_cold_wallets_address_blockchain_owner_id on cold_wallets using btree (owner_id, address, blockchain);
create index cold_wallets_blockchain_address_idx on cold_wallets using btree (blockchain, address);
create index cold_wallets_owner_id_idx on cold_wallets using btree (owner_id);
create index cold_wallets_created_at_idx on cold_wallets using btree (created_at);

-- Hot wallets
create table hot_wallets
(
  id                    uuid not null primary key default gen_random_uuid(),
  blockchain            varchar(255) not null check (blockchain != ''),
  address               varchar(255) not null check (address != ''),
  owner_id              uuid not null constraint fk_wallets_oid references owners,
  external_wallet_id    varchar(255) not null check (external_wallet_id != ''),
  sequence              int not null check (sequence >= 0),
  is_activated          bool not null default false,
  is_active             bool not null default true,
  is_dirty              bool not null default false,
  created_at            timestamp with time zone not null default (timezone('utc', now())),
  updated_at            timestamp with time zone
);
create unique index uni_idx_hot_wallets_blockchain_address_owner_id on hot_wallets using btree (owner_id, address, blockchain);
create index hot_wallets_blockchain_address_idx on hot_wallets using btree (blockchain, address);
create index hot_wallets_owner_id_idx on hot_wallets using btree (owner_id);
create index hot_wallets_created_at_idx on hot_wallets using btree (created_at);

-- Processing wallets
create table processing_wallets
(
  id          uuid not null primary key default gen_random_uuid(),
  blockchain  varchar(255) not null check (blockchain != ''),
  address     varchar(255) not null check (address != ''),
  owner_id    uuid not null constraint fk_wallets_oid references owners,
  sequence    int not null check (sequence >= 0),
  is_active   bool not null default true,
  is_dirty    bool not null default false,
  created_at  timestamp with time zone not null default (timezone('utc', now())),
  updated_at  timestamp with time zone
);
create unique index uni_idx_processing_wallets_blockchain_address_owner_id on processing_wallets using btree (owner_id, address, blockchain);
create index processing_wallets_blockchain_address_idx on processing_wallets using btree (blockchain, address);
create index processing_wallets_owner_id_idx on processing_wallets using btree (owner_id);
create index processing_wallets_created_at_idx on processing_wallets using btree (created_at);

-- Webhooks
create table webhooks (
  id            uuid not null primary key default gen_random_uuid(),
  kind          varchar(30) not null check (kind != ''),
  "status"      varchar(30) not null check ("status" != ''),
  attempts      int not null default 0,
  payload       jsonb not null default '{}',
  client_id     uuid not null constraint fk_webhooks_cid references clients,
  response      text null,
  created_at    timestamp with time zone not null default (timezone('utc', now())),
  sent_at       timestamp with time zone,
  updated_at    timestamp with time zone
);
create unique index webhooks_payload_client_id_uniq_idx on webhooks ((payload::text), client_id);
create index webhooks_status_idx on webhooks using btree (status);
create index webhooks_kind_idx on webhooks using btree (kind);
create index webhooks_attempts_idx on webhooks using btree (attempts);
create index webhooks_client_id_idx on webhooks using btree (client_id);
create index webhooks_created_at_idx on webhooks using btree (created_at);

CREATE OR REPLACE VIEW webhook_view AS (
  select w.*, c.callback_url, c.secret_key
  from webhooks w
  join clients c on c.id = w.client_id
);

-- Transfers
create table transfers (
  id                   uuid not null primary key default gen_random_uuid(),
  "status"             varchar(30) not null check ("status" != ''),
  client_id            uuid not null references clients,
  owner_id             uuid not null references owners,
  request_id           varchar(255) not null check (request_id != ''),
  blockchain           varchar(30) not null check (blockchain != ''),
  from_addresses       varchar(255)[] not null,
  to_addresses         varchar(255)[] not null,
  wallet_from_type     varchar(30) not null check (wallet_from_type != ''),
  asset_identifier     varchar(255) not null check (asset_identifier != ''),
  kind                 varchar(50) null,
  whole_amount         bool not null default false,
  amount               numeric(150,50),
  fee                  numeric(150,50),
  fee_max              numeric(150,50),
  tx_hash              varchar(100) null,
  created_at           timestamp with time zone not null default (timezone('utc', now())),
  updated_at           timestamp with time zone,
  state_data           jsonb not null default '{}',
  workflow_snapshot    jsonb not null default '{}',
  UNIQUE (client_id, owner_id, request_id)
);

create index transfers_status_idx on transfers using btree ("status");
create index transfers_owner_id_idx on transfers using btree (owner_id);
create index transfers_blockchain_idx on transfers using btree (blockchain);
create index transfers_request_id_idx on transfers using btree (request_id);
create index transfers_tx_hash_idx on transfers using btree (tx_hash);
create index transfers_created_at_idx on transfers using btree (created_at);

-- Processed blocks
create table processed_blocks
(
  blockchain  varchar(100) not null primary key check (blockchain != ''),
  number      bigint not null check (number >= 0),
  created_at  timestamp with time zone not null default (timezone('utc', now())),
  updated_at  timestamp with time zone,
  UNIQUE (blockchain)
);
