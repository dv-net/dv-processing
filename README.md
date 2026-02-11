<div align="center">

## ⚙️ DV Processing

<br>

[🌐 Website](https://dv.net) • [📖 Docs](https://docs.dv.net) • [🔌 API](https://docs.dv.net/en/operations/post-v1-external-wallet.html) • [💬 Support](https://dv.net/#support)

</div>

---

## 💡 Overview

**DV Processing** is the blockchain engine behind the DV.net ecosystem — a high-performance, multi-chain service that creates wallets, executes transfers, and tracks on-chain activity across 11 blockchains. Written in Go, it exposes a ConnectRPC/gRPC API and is designed to run on your own infrastructure.

> 🔒 **Non-custodial** — private keys and mnemonics never leave your server
>
> ⚡ **High-performance** — Go 1.24, ConnectRPC/gRPC, PostgreSQL, River queue
>
> 🌐 **Multi-chain** — 11 blockchains: EVM, UTXO, and Tron
>
> 🧱 **Modular** — FSM workflows, clean service layer, event-driven architecture

---

## ✨ Highlights

**🎯 Blockchain capabilities**
- ✅ Hot, cold, and processing wallet management per owner
- ✅ Transfer processing with per-chain finite state machines
- ✅ Tron resource delegation (energy & bandwidth) and activation contracts
- ✅ EVM gas price limits and fee control per network
- ✅ Bitcoin-family UTXO management with configurable fee-per-byte
- ✅ Block scanning and explorer proxy integration

**🔧 Technical features**
- ✅ ConnectRPC/gRPC API with request signing and authentication interceptors
- ✅ Background job processing via River queue
- ✅ Webhook delivery with retries, cleanup workers, and configurable TTL
- ✅ PostgreSQL storage with type-safe queries (`sqlc` + `pgxgen`)
- ✅ Protobuf-first API design with `buf` code generation
- ✅ Systemd deployment out of the box

---

## ⛓️ Supported Blockchains

| Chain | Type | Network |
|:------|:-----|:--------|
| Tron (TRX) | Account-based | mainnet |
| Ethereum (ETH) | EVM | mainnet / testnet |
| BNB Smart Chain (BSC) | EVM | mainnet / testnet |
| Polygon (MATIC) | EVM | mainnet / testnet |
| Arbitrum (ARB) | EVM | mainnet / testnet |
| Optimism (OP) | EVM | mainnet / testnet |
| Linea | EVM | mainnet / testnet |
| Bitcoin (BTC) | UTXO | mainnet / testnet |
| Litecoin (LTC) | UTXO | mainnet / testnet |
| Bitcoin Cash (BCH) | UTXO | mainnet / testnet |
| Dogecoin (DOGE) | UTXO | mainnet / testnet |

---

## 🧭 Architecture at a Glance

```text
cmd/                  CLI entrypoints (server, webhooks)
internal/
  handler/            ConnectRPC request handlers
  services/           Business logic (clients, owners, wallets, transfers, webhooks)
  fsm/                Finite state machines per blockchain (tron, evm, btc, ltc, bch, doge)
  workflow/           Workflow engine with stages, steps, and retries
  store/              PostgreSQL repositories (sqlc-generated)
  dispatcher/         Event dispatching
  taskmanager/        Background jobs (River)
  eproxy/             Explorer proxy client
  rmanager/           Resource manager (Tron delegation)
  watcher/            Blockchain watcher integration
  tscanner/           Transfer scanner
  escanner/           Explorer scanner
pkg/
  walletsdk/          Blockchain wallet SDKs (btc, ltc, bch, doge, evm, tron)
  postgres/           Database connection management
  encryption/         Encryption utilities
  retry/              Retry policies
schema/               Protobuf service definitions
sql/                  Migrations and SQL queries
artifacts/            Deployment configs (systemd, scripts)
```

---

## 🚀 Getting Started

**Build from source**
```bash
git clone https://github.com/dv-net/dv-processing.git
cd dv-processing

make build
```

The binary will appear at `bin/processing`.

**Run locally**
```bash
cp config.template.yaml config.yaml
# edit config.yaml with your database and node settings

make migrate up
make start
```

> ℹ️ Full deployment guide and Docker Compose setup are available in the [`dv-bundle`](https://github.com/dv-net/dv-bundle) repo and at https://docs.dv.net.

---

## 🛠 CLI Commands

- `dv-processing start` — start the gRPC/ConnectRPC server.
- `dv-processing migrate` — run database migrations (up / down / drop).
- `dv-processing blockchain` — blockchain tools (e.g. tron reclaim-resource).
- `dv-processing config` — validate config, generate envs and flags.
- `dv-processing utils` — utilities (systemd install, readme generation).
- `dv-processing version` — print the current version.

**💡 Example — reclaim Tron resources**
```bash
./dv-processing blockchain tron reclaim-resource \
  -pa TQ6DkBmxz3Zk7neh8mwmmkfJsVjrE9wwjY \
  -da TAoG3QdbgZ7saGBHiXVHRgdNJVpwUGqZZh \
  -type bandwidth
```

---

## 🧪 Development & Testing

**📦 Install dev tools**
```bash
make install-dev-tools
```

**⚙️ Code generation**
```bash
make gen          # run all generators (sql, proto, envs, abi)
make gensql       # regenerate sqlc queries
make genproto     # regenerate protobuf & ConnectRPC stubs
make genenvs      # regenerate environment variable docs
```

**🔍 Linting & formatting**
```bash
make lint
make fmt
```

**🔄 Live reload**
```bash
make watch        # uses air for hot reloading
```

---

## 📋 Configuration

The service is configured via `config.yaml` (see `config.template.yaml`) and/or environment variables.

All environment variables are prefixed with `PROCESSING_` and follow this structure:

| | Category | Prefix | Examples |
|:--|:---------|:-------|:---------|
| 📝 | Logging | `PROCESSING_LOG_*` | `PROCESSING_LOG_LEVEL`, `PROCESSING_LOG_FORMAT` |
| 📊 | Ops / Monitoring | `PROCESSING_OPS_*` | `PROCESSING_OPS_METRICS_ENABLED`, `PROCESSING_OPS_HEALTHY_ENABLED` |
| 🗄️ | Database | `PROCESSING_POSTGRES_*` | `PROCESSING_POSTGRES_ADDR`, `PROCESSING_POSTGRES_DB_NAME` |
| 🔌 | gRPC Server | `PROCESSING_GRPC_*` | `PROCESSING_GRPC_ADDR`, `PROCESSING_GRPC_REFLECT_ENABLED` |
| ⛓️ | Blockchains | `PROCESSING_BLOCKCHAIN_*` | `PROCESSING_BLOCKCHAIN_TRON_ENABLED` |
| 🔔 | Webhooks | `PROCESSING_WEBHOOKS_*` | `PROCESSING_WEBHOOKS_SENDER_ENABLED` |
| 💸 | Transfers | `PROCESSING_TRANSFERS_*` | `PROCESSING_TRANSFERS_ENABLED` |

> ℹ️ Full environment variable reference is auto-generated via `make genenvs`.

---

## 📦 Deployment

**Systemd**
```bash
./dv-processing utils systemd
```

This generates and installs a systemd unit file for production deployments on Linux.

---

## 🔐 Security

1. 🔓 **Non-custodial** — mnemonics and private keys are encrypted at rest and never leave the server.
2. ✍️ **Request signing** — all API calls are authenticated via signature-based interceptors.
3. 🛡️ **Two-factor authentication** — TOTP-based 2FA for sensitive owner operations.
4. 🔑 **Encrypted storage** — mnemonics and OTP secrets are encrypted in the database.

---

## 📡 API Services

The ConnectRPC API exposes the following services on port `9000`:

| | Service | Description |
|:--|:--------|:------------|
| 👤 | `ClientService` | Merchant/client management and callback URLs |
| 🏠 | `OwnerService` | Owner creation, mnemonic management, 2FA |
| 💳 | `WalletService` | Hot, cold, and processing wallet operations |
| 💸 | `TransferService` | Transfer creation and status tracking |
| ⚙️ | `SystemService` | System info, version checking, logs |

Proto definitions are located in `schema/processing/` and compiled with `buf`.

---

## 🤝 Contributing

```bash
# Before submitting a PR
make lint
make fmt
go test ./...
```

- ⭐ Star the repo if it helps your project.
- 🐛 Report bugs via Issues.
- 💡 Propose new features and use cases.
- 🔧 Pull Requests are welcome!

---

## 💝 Donations

Support the development of the project with crypto:

> <img src="docs/assets/icons/coins/IconUsdt.png" width="17"> **USDT (Tron)** — `TCB4bYYN5x1z9Z4bBZ7p3XxcMwdtCfmNdN`

> <img src="docs/assets/icons/coins/IconBtcBitcoin.png" width="17"> **Bitcoin** — `bc1qemvhkgzr4r7ksgxl8lv0lw7mnnthfc6990v3c2`

> <img src="docs/assets/icons/coins/IconTrxTron.png" width="17"> **TRON (TRX)** — `TCB4bYYN5x1z9Z4bBZ7p3XxcMwdtCfmNdN`

> <img src="docs/assets/icons/coins/IconEthEthereum.png" width="17"> **Ethereum** — `0xf1e4c7b968a20aae891cc18b1d5836b806691d47`

🔗 Other networks and tokens (BNB Chain, Arbitrum, Polygon, Litecoin, Dogecoin, Bitcoin Cash, etc.) are available at **[payment form](https://cloud.dv.net/pay/store/208ec77f-d516-46b9-b280-3c12e1a75071/donate)**

---

## 📞 Contact

<div align="center">

**Telegram:** [@dv_net_support_bot](https://t.me/dv_net_support_bot) • **Telegram Chat:** [@dv_net_support_chat](https://t.me/dv_net_support_chat) • **Discord:** [discord.gg/Szy2XGsr](https://discord.gg/Szy2XGsr)

**Email:** [support@dv.net](https://dv.net/#support) • **Website:** [dv.net](https://dv.net) • **Documentation:** [docs.dv.net](https://docs.dv.net)

</div>

---

<div align="center">

**© 2025 DV.net** • [DV Technologies Ltd.](https://dv.net)

*Built with ❤️ for the crypto community*

</div>
