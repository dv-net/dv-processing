-- VIEWS
drop view webhook_view;

-- TABLES
drop table if exists processed_blocks CASCADE;
drop table if exists transfers CASCADE;
drop table if exists webhooks CASCADE;
drop table if exists cold_wallets CASCADE;
drop table if exists hot_wallets CASCADE;
drop table if exists processing_wallets CASCADE;
drop table if exists owners CASCADE;
drop table if exists clients CASCADE;
