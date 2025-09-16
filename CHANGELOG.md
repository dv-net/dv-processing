# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog]https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning]https://semver.org/spec/v2.0.0.html).

## Releases

### Unreleased
- Add memory buffer for processing logs [DV-3361]

### [0.9.4] - 2025-09-15
- Sign packages with GPG [DV-3298]
- Reworked hot wallet key retrieval to simplify address handling [DV-3142]
- Preserve logging on appropriate levels based on RPC code returned from service [DV-3337]
- Reduce log levels across services to debug for less verbosity [DV-3350]

### [0.6.6] - 2025-07-17

- Added retries on network error on resource manager requests [DV-2783]
- Added owner confirmation and custom mnemonic [DV-2722]

### [0.6.5] - 2025-07-14

- Dogecoin enabled by default [DV-2547]
- EVM fix transfers [DV-2008]
- Arbitrum validation fix [DV-2718]

### [0.6.4] - 2025-07-03

- Skip network request to resource manager if no operation is required [DV-2434]
- Integrated new version of resource manager with activation, reservation [DV-2241]
- Fixed burn activation estimation [DV-2558]
- Dogecoin integration [DV-2532]
- Additional filtering for hot wallet private keys retrieval [DV-2597]
- Fix creating processing wallets [DV-2617]
- Fix withdrawl USDT from processing wallets [DV-2592]
- Tron fsm compensation stage added [DV-2610]

### [0.6.3] - 2025-06-17

- Transfer transactions persist [DV-2031]
- Added arbitrum, optimism, linea [DV-2121]

### [0.6.2] - 2025-05-15

- Encrypt seed phrase in gzip by owner_id [DV-1906]
- Use resources for processing wallets [DV-1986]
- Fix external Tron lib [DV-1993]

### [0.6.1] - 2025-05-12

- Fix EVM transfer [DV-1878]
- Introduce Polygon [DV-1828]
- Improved commissions for EVM transfers [DV-1900]
- Watcher v2 [DV-1875]

### [0.6.0] - 2025-04-22

- Introduce Binance Smart Chain [DV-1390]
- Use the same evm address for the same external id [DV-1645]
- Skip zero transfers [DV-1797]
- Improve creating the same evm hot wallets [DV-1831]

### [0.5.4] - 2025-04-10

- Fix tron trx transfer to not activated address [DV-1649]
- Fix health check for resource manager [DV-1689]
- Added system contract activation (CreateAccount) [DV-701]

### [0.5.3] - 2025-04-01

- Improved processing wallet withdrawals [DV-1604]

### [0.5.2] - 2025-03-28

- Added stable repo publish [DV-955]
- Fix transfers with multiple utxo for BTC and LTC [DV-1370]
- Fix estimate tx fee for BTC, LTC and BCH [DV-1210]
- Update processing from package manager [DV-1199]
- Fix processing activating addresses via smart-contract estimation [DV-1454]
- Webhook status fix [DV-1510]

### [0.5.1] - 2025-03-07

- Fix tron cancel resources reservation

### [0.5.0] - 2025-03-04

- 🚨 BREAKING: Implement WalletSDK with BIP for BTC, LTC, ETH and TRON [DV-1168]
- 🚨 BREAKING: Make passphrase optional [DV-1268]

### [0.4.0] - 2025-02-21

- Added support for new flow of reservation and activation [DV-1052]
- Introduce Bitcoin Cash and key value explorers support [DV-1058]
- Added webhook system check for activation deposits [DV-1084]
- Moved TRON resource manager availability check into transfer request
  validation [DV-1053]
- Integrated new resource manager's version with activation, reservation [DV-1052]
- Remove transaction type from webhook to backend [DV-700]
- Backend webhooks fsm steps [DV-639]
- Added support for new version of resource manager [DV-994]
- Improve ETH Transfers [DV-756]
- Prevent unit-file replacement [DV-1120]

### [0.3.1] - 2025-01-23

- Litecoin watcher integration [DV-457]
- Download private keys with filters [DV-460]
- Total used energy and bandwidth info for tron wallet [DV-776]
- Admin auth rework [DV-1011]

### [0.3.0] - 2025-01-17

- Move to home dir [DV-697]
- Move stage server [DV-561]
- Move dev server [DV-560]
- Fix ETH Transfers [DV-656]
- Non-root executable [DV-57]

### [0.2.7] - 2024-12-17

- Added merchant external registration in admin service logic [DV-121]
- Added TRX stacked resources information retrieval [DV-53]
- Fix tron system webhook for burning trx [DV-444]
- Update handling error in fsm for LTC and BTC [DV-477]
- Processing identity added for external apis [DV-378]
- Add tiny request processing wallets [DV-304]
- ETH check transfer in mempool [DV-511]
- Watcher ping [DV-578]

### [0.2.6] - 2024-12-04

- Bitcoin watcher integration [DV-227]
- Litecoin integration [DV-216]

### [0.2.5] - 2024-11-28

- BTC multiple inputs in one transfer [DV-146]
- Added retrier for some grpc requests [DV-178]
- ETH Transfer optimization [DV-237]
- ETH - use etimated data while transfer [DV-265]
- ETH reservation logic [DV-222]
- Use traefik btc endpoint by default [DV-115]

### [0.2.4] - 2024-11-11

- Fix tron account resource insufficient error [DV-150]
- Save more transfer info to state data [DV-149]
- Take into account the bandwidth to activate the wallet during the transfer [DV-151]

### [0.2.3] - 2024-11-08

- Changes added to System API [DV-2587]
- Remove go mod tidy from build [DV-2265]
- Rsync logs as defaults [DV-2409]
- Method for get URL added [DV-2599]
- BTC fix webhook [DV-2664]
- Fix reclaim bandwidth [DV-2583]
- Fix reclaim energy [DV-2701]
- CLI tool for reclaim tron resources [DV-2721]
- Fix checking processing response code [DV-2723]
- ETH config max gas limit [DV-2564]
- Fix some failed transfers and deposits in btc and tron [DV-2751]

### [0.2.2] - 2024-10-18

- ETH Transfers [DV-2090]
- Fix TRON resources info [DV-2393]
- RPC method for system info added [DV-2375]
- Validator pkg dependency from internal resolved [DV-2457]

### [0.2.1] - 2024-10-11

- Improve transfers API and TRON delegation resources [DV-2383]

### [0.2.0] - 2024-10-03

- TRON transfers [DV-2152]
- Withdraw native coin using hot wallet [DV-2318]
- Checking address from when creating a transfer [DV-2310]
- Tron wallet activation bug [DV-2314]
- System webhooks [DV-2305]
- Priority for withdrawal of contracts and native coin [DV-2324]
- Standardize rpc error codes [DV-2332]

### [0.1.4] - 2024-09-30

- Hot wallet balances panic fixed [DV-2261]
- Go releaser fixes [DV-1755]

### [0.1.3] - 2024-09-26

- Added go-releaser [DV-2226]
- Added command to generate default configuration [DV-2217]
- Add cron cleanup for old webhooks [DV-1944]
- Add two-factor token validation API method [DV-2171]
- BTC transfers and FSM improvements [DV-2137]
- Changed use default configuration to implicit [DV-2232]

### [0.1.2] - 2024-09-09

- BTC transfers [DV-2089]
- Expanded protobuf API providing information for wallet processing [DV-2127]
- Transfer contract, API for transferring and storing transfer requests in the
  database [DV-2086]
- Added 2FA to AttachOwnerColdWallet [DV-2132]
- Update wallets whitelist API [DV-2131]

### [0.1.1] - 2024-08-26

- Add check exists owners external identifier [DV-1975]
- Check sign in API requests [DV-1806]
- Webhook duplicate constraint fix [DV-2066]
- Change webhook retry policy for blockchains [DV-2067]

### [0.1.0] - 2024-08-21

- Deposits for TRON [DV-1931]
- Deposits for BTC [DV-1932]
- Deposits for ETH [DV-1933]
- Refactor Withdrawl table [DV-1824]
- Remove blocks stream parsing [DV-1804]
- Added fixes to two-factor [DV-2005]
- JSON rest api in snake case [DV-2002]
- Processing big refactoring [DV-1985]
- Added River task manager [DV-2041]

### [0.0.2] - 2024-08-13

- Add dev ci-cd [DV-1899]
- Add ETH, TRX wallet generation [DV-1795]
- Fixed 2fa caching [DV-1808]
- Refactored 2FA [DV-1918]
- Refactored protobuf scheme [DV-1839]
- Reworked GetOwnerPrivateKeys API method [DV-1846]

### [0.0.1] - 2024-07-01

- Initial commit
