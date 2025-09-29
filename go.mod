module github.com/dv-net/dv-processing

go 1.24.2

require (
	connectrpc.com/connect v1.18.1
	connectrpc.com/cors v0.1.0
	github.com/btcsuite/btcd v0.24.3-0.20250213152832-bb52d7d78d9c
	github.com/btcsuite/btcd/btcec/v2 v2.3.5-0.20250213152832-bb52d7d78d9c
	github.com/btcsuite/btcd/btcutil v1.1.7-0.20250213152832-bb52d7d78d9c
	github.com/btcsuite/btcd/chaincfg/chainhash v1.1.1-0.20250213152832-bb52d7d78d9c
	github.com/decred/dcrd/dcrec/secp256k1/v4 v4.4.0
	github.com/ethereum/go-ethereum v1.15.8
	github.com/fbsobreira/gotron-sdk v0.0.0-20250427130616-96b87f5d2100
	github.com/go-playground/validator/v10 v10.26.0
	github.com/goccy/go-json v0.10.5
	github.com/golang-migrate/migrate/v4 v4.18.2
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.4
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pquerna/otp v1.4.0
	github.com/prometheus/client_golang v1.22.0
	github.com/rs/cors v1.11.1
	github.com/shopspring/decimal v1.4.0
	github.com/stretchr/testify v1.10.0
	github.com/urfave/cli/v2 v2.27.6
	go.akshayshah.org/connectproto v0.6.0
	golang.org/x/net v0.41.0
	google.golang.org/protobuf v1.36.7
)

require (
	connectrpc.com/grpcreflect v1.3.0 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/bits-and-blooms/bitset v1.22.0 // indirect
	github.com/btcsuite/go-socks v0.0.0-20170105172521-4720035b7bfd // indirect
	github.com/btcsuite/websocket v0.0.0-20150119174127-31079b680792 // indirect
	github.com/consensys/bavard v0.1.30 // indirect
	github.com/consensys/gnark-crypto v0.17.0 // indirect
	github.com/crate-crypto/go-ipa v0.0.0-20240724233137-53bbb0ceb27a // indirect
	github.com/crate-crypto/go-kzg-4844 v1.1.0 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/deckarep/golang-set v1.8.0 // indirect
	github.com/deckarep/golang-set/v2 v2.8.0 // indirect
	github.com/dv-net/mx v0.1.1
	github.com/ethereum/c-kzg-4844 v1.0.3 // indirect
	github.com/ethereum/go-verkle v0.2.2 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gcash/bchlog v0.0.0-20180913005452-b4f036f92fa6 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.27.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.10 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mmcloughlin/addchain v0.4.0 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/riverqueue/river/riverdriver v0.22.0 // indirect
	github.com/riverqueue/river/rivershared v0.22.0 // indirect
	github.com/riverqueue/river/rivertype v0.22.0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/rjeczalik/notify v0.9.3 // indirect
	github.com/shirou/gopsutil v3.21.11+incompatible // indirect
	github.com/supranational/blst v0.3.14 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/tklauser/go-sysconf v0.3.15 // indirect
	github.com/tklauser/numcpus v0.10.0 // indirect
	github.com/tyler-smith/go-bip39 v1.1.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opentelemetry.io/auto/sdk v1.1.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	golang.org/x/exp v0.0.0-20250228200357-dead58393ab7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250818200422-3122310a409c // indirect
	lukechampine.com/blake3 v1.4.0 // indirect
	rsc.io/tmplfunc v0.0.3 // indirect
)

require (
	github.com/dv-net/mx/clients/connectrpc_client v0.0.0-20250826134925-03bc3492dba3
	github.com/dv-net/mx/transport/connectrpc_transport v0.0.0-20250826134925-03bc3492dba3
	github.com/dv-net/xconfig/decoders/xconfigyaml v0.0.0-20250828100326-2c7d793ffc71
	github.com/gcash/bchd v0.20.0
	github.com/gcash/bchutil v0.0.0-20250115071209-216bd54f0d4d
	github.com/gobeam/stringy v0.0.7
	github.com/goccy/go-yaml v1.18.0
	github.com/jellydator/ttlcache/v3 v3.3.0
	github.com/ltcsuite/ltcd v0.23.5
	github.com/ltcsuite/ltcd/btcec/v2 v2.3.2
	github.com/ltcsuite/ltcd/chaincfg/chainhash v1.0.2
	github.com/ltcsuite/ltcd/ltcutil v1.1.3
	github.com/puzpuzpuz/xsync/v4 v4.0.0
	github.com/riverqueue/river v0.22.0
	github.com/riverqueue/river/riverdriver/riverpgxv5 v0.22.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/samber/lo v1.50.0
	github.com/tidwall/gjson v1.18.0
	github.com/urfave/cli/v3 v3.3.1
	google.golang.org/grpc v1.75.0
)

require (
	github.com/dv-net/go-bip39 v1.1.1
	github.com/dv-net/xconfig v0.1.0
)

require github.com/dv-net/dv-proto v0.5.5

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/boombuler/barcode v1.0.2 // indirect
	github.com/btcsuite/btclog v0.0.0-20241017175713-3428138b75c7 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/cpuguy83/go-md2man/v2 v2.0.6 // indirect
	github.com/cristalhq/aconfig v0.18.6 // indirect
	github.com/cristalhq/aconfig/aconfigdotenv v0.17.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/decred/dcrd/crypto/blake256 v1.1.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/georgysavva/scany/v2 v2.1.4
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/holiman/uint256 v1.3.2 // indirect
	github.com/huandu/go-sqlbuilder v1.35.0
	github.com/huandu/xstrings v1.5.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/client_model v0.6.2 // indirect
	github.com/prometheus/common v0.63.0 // indirect
	github.com/prometheus/procfs v0.16.0 // indirect
	github.com/russross/blackfriday/v2 v2.1.0 // indirect
	github.com/shengdoushi/base58 v1.0.0 // indirect
	github.com/xrash/smetrics v0.0.0-20240521201337-686a1a2994c1 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.60.0 // indirect
	go.opentelemetry.io/otel v1.37.0 // indirect
	go.opentelemetry.io/otel/metric v1.37.0 // indirect
	go.opentelemetry.io/otel/trace v1.37.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.40.0
	golang.org/x/sync v0.16.0
	golang.org/x/sys v0.34.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20250818200422-3122310a409c // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
