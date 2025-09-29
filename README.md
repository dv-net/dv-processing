# DV PROCESSING

## Usage

```text
NAME:
   dv-processing - A new cli application

USAGE:
   dv-processing [global options] [command [command options]]

VERSION:
   local-unknown

COMMANDS:
   config      validate, gen envs and flags for config
   start       start the server
   migrate     migration database schema
   blockchain  blockchain tools
   utils       custom cli utils
   version     print the version
   help, h     Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

## Environments

| **Name**                                                             | **Required** | **Secret** | **Default value**                    | **Usage**                                                                  | **Example**                 |
|----------------------------------------------------------------------|--------------|------------|--------------------------------------|----------------------------------------------------------------------------|-----------------------------|
| `PROCESSING_LOG_FORMAT`                                              |              |            | `json`                               | allows to set custom formatting                                            | `json`                      |
| `PROCESSING_LOG_LEVEL`                                               |              |            | `info`                               | allows to set custom logger level                                          | `info`                      |
| `PROCESSING_LOG_CONSOLE_COLORED`                                     |              |            | `false`                              | allows to set colored console output                                       | `false`                     |
| `PROCESSING_LOG_TRACE`                                               |              |            | `fatal`                              | allows to set custom trace level                                           | `fatal`                     |
| `PROCESSING_LOG_WITH_CALLER`                                         |              |            | `false`                              | allows to show caller                                                      | `false`                     |
| `PROCESSING_LOG_WITH_STACK_TRACE`                                    |              |            | `false`                              | allows to show stack trace                                                 | `false`                     |
| `PROCESSING_OPS_ENABLED`                                             |              |            | `false`                              | allows to enable ops server                                                | `false`                     |
| `PROCESSING_OPS_NETWORK`                                             | ✅            |            | `tcp`                                | allows to set ops listen network: tcp/udp                                  | `tcp`                       |
| `PROCESSING_OPS_TRACING_ENABLED`                                     |              |            | `false`                              | allows to enable tracing                                                   | `false`                     |
| `PROCESSING_OPS_METRICS_ENABLED`                                     |              |            | `false`                              | allows to enable metrics                                                   | `true`                      |
| `PROCESSING_OPS_METRICS_PATH`                                        | ✅            |            | `/metrics`                           | allows to set custom metrics path                                          | `/metrics`                  |
| `PROCESSING_OPS_METRICS_PORT`                                        | ✅            |            | `10000`                              | allows to set custom metrics port                                          | `10000`                     |
| `PROCESSING_OPS_METRICS_BASIC_AUTH_ENABLED`                          |              |            | `false`                              | allows to enable basic auth                                                |                             |
| `PROCESSING_OPS_METRICS_BASIC_AUTH_USERNAME`                         |              |            |                                      | auth username                                                              |                             |
| `PROCESSING_OPS_METRICS_BASIC_AUTH_PASSWORD`                         |              |            |                                      | auth password                                                              |                             |
| `PROCESSING_OPS_HEALTHY_ENABLED`                                     |              |            | `false`                              | allows to enable health checker                                            | `true`                      |
| `PROCESSING_OPS_HEALTHY_PATH`                                        | ✅            |            | `/healthy`                           | allows to set custom healthy path                                          | `/healthy`                  |
| `PROCESSING_OPS_HEALTHY_PORT`                                        | ✅            |            | `10000`                              | allows to set custom healthy port                                          | `10000`                     |
| `PROCESSING_OPS_PROFILER_ENABLED`                                    |              |            | `false`                              | allows to enable profiler                                                  | `false`                     |
| `PROCESSING_OPS_PROFILER_PATH`                                       | ✅            |            | `/debug/pprof`                       | allows to set custom profiler path                                         | `/debug/pprof`              |
| `PROCESSING_OPS_PROFILER_PORT`                                       | ✅            |            | `10000`                              | allows to set custom profiler port                                         | `10000`                     |
| `PROCESSING_OPS_PROFILER_WRITE_TIMEOUT`                              |              |            | `60`                                 | HTTP server write timeout in seconds                                       | `60`                        |
| `PROCESSING_POSTGRES_ADDR`                                           | ✅            |            |                                      |                                                                            | `localhost:5432`            |
| `PROCESSING_POSTGRES_USER`                                           | ✅            | ✅          |                                      |                                                                            |                             |
| `PROCESSING_POSTGRES_PASSWORD`                                       | ✅            | ✅          |                                      |                                                                            |                             |
| `PROCESSING_POSTGRES_DB_NAME`                                        | ✅            |            |                                      |                                                                            |                             |
| `PROCESSING_POSTGRES_PING_INTERVAL`                                  | ✅            |            | `10`                                 |                                                                            |                             |
| `PROCESSING_POSTGRES_MIN_CONNS`                                      | ✅            |            | `3`                                  |                                                                            |                             |
| `PROCESSING_POSTGRES_MAX_CONNS`                                      | ✅            |            | `6`                                  |                                                                            |                             |
| `PROCESSING_GRPC_ENABLED`                                            |              |            | `true`                               | allows to enable server                                                    | `true`                      |
| `PROCESSING_GRPC_ADDR`                                               | ✅            |            | `:9000`                              | server listen address                                                      | `localhost:9000`            |
| `PROCESSING_GRPC_REFLECT_ENABLED`                                    |              |            | `false`                              | allows to enable reflection service                                        | `false`                     |
| `PROCESSING_EXPLORER_PROXY_NAME`                                     | ✅            |            | `explorer-proxy-client`              |                                                                            | `backend-connectrpc-client` |
| `PROCESSING_EXPLORER_PROXY_ADDR`                                     |              |            | `https://explorer-proxy.dv.net`      | connectrpc server address                                                  | `localhost:9000`            |
| `PROCESSING_BLOCKCHAIN_TRON_ENABLED`                                 |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_TRON_NODE_GRPC_ADDRESS`                       |              |            | `node-tron-grpc.dv.net:443`          | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_TRON_NODE_USE_TLS`                            |              |            | `true`                               | use tls                                                                    |                             |
| `PROCESSING_BLOCKCHAIN_TRON_ACTIVATION_CONTRACT_ADDRESS`             |              |            | `TQuCVz7ZXMwcuT2ERcBYCZzLeNAZofcTgY` |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_TRON_USE_BURN_TRX_ACTIVATION`                 |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_ETHEREUM_ENABLED`                             |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_ETHEREUM_NETWORK`                             | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_ETHEREUM_NODE_ADDRESS`                        |              |            | `https://node-eth.dv.net`            | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_ETHEREUM_ATTRIBUTES_MAX_GAS_PRICE`            |              |            | `8`                                  | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_BINANCE_SMART_CHAIN_ENABLED`                  |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BINANCE_SMART_CHAIN_NETWORK`                  | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_BINANCE_SMART_CHAIN_NODE_ADDRESS`             |              |            | `https://node-bsc.dv.net`            | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_BINANCE_SMART_CHAIN_ATTRIBUTES_MAX_GAS_PRICE` |              |            | `3`                                  | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_POLYGON_ENABLED`                              |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_POLYGON_NETWORK`                              | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_POLYGON_NODE_ADDRESS`                         |              |            | `https://node-polygon.dv.net`        | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_POLYGON_ATTRIBUTES_MAX_GAS_PRICE`             |              |            | `130`                                | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_ARBITRUM_ENABLED`                             |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_ARBITRUM_NETWORK`                             | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_ARBITRUM_NODE_ADDRESS`                        |              |            | `https://node-arbitrum.dv.net`       | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_ARBITRUM_ATTRIBUTES_MAX_GAS_PRICE`            |              |            | `1`                                  | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_OPTIMISM_ENABLED`                             |              |            | `false`                              |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_OPTIMISM_NETWORK`                             | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_OPTIMISM_NODE_ADDRESS`                        |              |            |                                      | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_OPTIMISM_ATTRIBUTES_MAX_GAS_PRICE`            |              |            | `0.001`                              | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_LINEA_ENABLED`                                |              |            | `false`                              |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_LINEA_NETWORK`                                | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_LINEA_NODE_ADDRESS`                           |              |            |                                      | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_LINEA_ATTRIBUTES_MAX_GAS_PRICE`               |              |            | `1`                                  | max gas price in Gwei                                                      |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_ENABLED`                              |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_NETWORK`                              | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_BITCOIN_ATTRIBUTES_FEE_PER_BYTE`              |              |            | `7`                                  |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_ATTRIBUTES_MIN_UTXO_AMOUNT`           |              |            | `0`                                  | min UTXO amount in satoshi                                                 |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_NODE_ADDRESS`                         |              |            | `node-btc.dv.net:443`                | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_NODE_LOGIN`                           |              | ✅          |                                      | node login                                                                 |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_NODE_SECRET`                          |              | ✅          |                                      |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_NODE_USE_TLS`                         |              |            | `true`                               | use TLS for connection                                                     |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_ENABLED`                             |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_NETWORK`                             | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_LITECOIN_ATTRIBUTES_FEE_PER_BYTE`             |              |            | `10`                                 |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_ATTRIBUTES_MIN_UTXO_AMOUNT`          |              |            | `0`                                  | min UTXO amount in satoshi                                                 |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_NODE_ADDRESS`                        |              |            | `node-ltc.dv.net`                    | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_NODE_LOGIN`                          |              | ✅          |                                      | node login                                                                 |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_NODE_SECRET`                         |              | ✅          |                                      |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_LITECOIN_NODE_USE_TLS`                        |              |            | `true`                               | use TLS for connection                                                     |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_ENABLED`                         |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_NETWORK`                         | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_ATTRIBUTES_FEE_PER_BYTE`         |              |            | `1`                                  |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_ATTRIBUTES_MIN_UTXO_AMOUNT`      |              |            | `0`                                  | min UTXO amount in satoshi                                                 |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_NODE_ADDRESS`                    |              |            | `node-bch.dv.net`                    | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_NODE_LOGIN`                      |              | ✅          |                                      | node login                                                                 |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_NODE_SECRET`                     |              | ✅          |                                      |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_BITCOIN_CASH_NODE_USE_TLS`                    |              |            | `true`                               | use TLS for connection                                                     |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_ENABLED`                             |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_NETWORK`                             | ✅            |            | `mainnet`                            |                                                                            | `mainnet / testnet`         |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_ATTRIBUTES_FEE_PER_BYTE`             |              |            | `50000`                              |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_ATTRIBUTES_MIN_UTXO_AMOUNT`          |              |            | `0`                                  | min UTXO amount in satoshi                                                 |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_NODE_ADDRESS`                        |              |            | `node-doge.dv.net:443`               | node address                                                               |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_NODE_LOGIN`                          |              | ✅          |                                      | node login                                                                 |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_NODE_SECRET`                         |              | ✅          |                                      |                                                                            |                             |
| `PROCESSING_BLOCKCHAIN_DOGECOIN_NODE_USE_TLS`                        |              |            | `true`                               | use TLS for connection                                                     |                             |
| `PROCESSING_RESOURCE_MANAGER_ENABLED`                                |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_RESOURCE_MANAGER_CONNECT_NAME`                           | ✅            |            | `orders-client`                      |                                                                            | `backend-connectrpc-client` |
| `PROCESSING_RESOURCE_MANAGER_CONNECT_ADDR`                           |              |            | `https://delegate.dv.net`            | connectrpc server address                                                  | `localhost:9000`            |
| `PROCESSING_RESOURCE_MANAGER_USE_ACTIVATOR_CONTRACT`                 |              |            | `true`                               |                                                                            |                             |
| `PROCESSING_RESOURCE_MANAGER_DELEGATION_DURATION`                    |              |            | `15s`                                |                                                                            | `60s/1m/1h`                 |
| `PROCESSING_WATCHER_CLIENT_SECRET`                                   |              |            |                                      |                                                                            |                             |
| `PROCESSING_WATCHER_GRPC_RECONNECTION_DELAY`                         |              |            | `1s`                                 |                                                                            | `1s`                        |
| `PROCESSING_WATCHER_ENABLED`                                         |              |            | `false`                              |                                                                            |                             |
| `PROCESSING_WATCHER_CONNECT_NAME`                                    | ✅            |            | `connectrpc-client`                  |                                                                            | `backend-connectrpc-client` |
| `PROCESSING_WATCHER_CONNECT_ADDR`                                    |              |            |                                      | connectrpc server address                                                  | `localhost:9000`            |
| `PROCESSING_WEBHOOKS_SENDER_ENABLED`                                 |              |            | `true`                               | allows to enable and disable webhook sender                                | `true / false`              |
| `PROCESSING_WEBHOOKS_SENDER_QUANTITY`                                |              |            | `500`                                | allows you to specify the number of webhooks to send at once               | `500`                       |
| `PROCESSING_WEBHOOKS_CLEANUP_ENABLED`                                |              |            | `true`                               | allows to enable and disable webhook cleanup worker                        | `true / false`              |
| `PROCESSING_WEBHOOKS_CLEANUP_CRON`                                   |              |            | `0 1 * * *`                          | allows to set custom cron rule for cleanup old competed webhooks           | `0 1 * * *`                 |
| `PROCESSING_WEBHOOKS_CLEANUP_MAX_AGE`                                |              |            | `720h0m0s`                           | allows to set max age for completed webhooks. by default it 720h (30 days) | `720h`                      |
| `PROCESSING_WEBHOOKS_REMOVE_AFTER_SENT`                              |              |            | `false`                              | allows to remove a webhook after it has been successfully sent             | `true / false`              |
| `PROCESSING_INTERCEPTORS_DISABLE_CHECKING_SIGN`                      |              |            | `false`                              | allows to disable checking sign of request                                 | `true / false`              |
| `PROCESSING_TRANSFERS_ENABLED`                                       |              |            | `true`                               | allows to enable transfers service                                         | `true / false`              |
| `PROCESSING_USE_CACHE_FOR_WALLETS`                                   |              |            | `true`                               | allows to use cache for wallets. this option is experimental               | `true / false`              |
| `PROCESSING_MERCHANT_ADMIN_BASE_URL`                                 |              |            | `https://api.dv.net`                 | allows to set merchant admin service endpoint                              | `https://api.dv.net`        |
| `PROCESSING_UPDATER_BASE_URL`                                        |              |            | `http://localhost:8081`              | allows to set updater service endpoint                                     | `http://localhost:8081`     |

## Install Systemd service

Use the following command to install the systemd service

```bash
./dv_processing utils systemd
```

## Addresses for tests

```text
Tron Bybit wallet: TU4vEruvZwLLkSfV9bNw12EJTPvNr7Pvaa
Ethereum Coinbase cold wallet: 0xb5d85cbf7cb3ee0d56b3bb207d5fc4b82f43f511
Polygon polymarket: 0xa7fd99748ce527eadc0bdac60cba8a4ef4090f7c
Litecoin ltc1qpkme9t7h3sd47ku3mlyrysqwf5uy9xy52yqss4
Bitcoin: bc1qryhgpmfv03qjhhp2dj8nw8g4ewg08jzmgy3cyx
Bitcoin cash: qp028nlln35nwnv5a9dssw9w57z5n765rgenr3suw6
```
