//nolint:goconst
package config

import (
	"errors"
	"time"

	"github.com/dv-net/dv-processing/pkg/postgres"
	"github.com/dv-net/mx/clients/connectrpc_client"
	"github.com/dv-net/mx/logger"
	"github.com/dv-net/mx/ops"
	"github.com/dv-net/mx/transport/connectrpc_transport"
)

type Config struct {
	Log             logger.Config
	Ops             ops.Config
	Postgres        postgres.Config
	Grpc            connectrpc_transport.Config
	ExplorerProxy   connectrpc_client.Config `yaml:"explorer_proxy"`
	Blockchain      Blockchain
	ResourceManager ResourceManager `yaml:"resource_manager"`
	Watcher         Watcher         `yaml:"watcher"`
	Webhooks        struct {
		Sender struct {
			Enabled  bool  `json:"enabled" yaml:"enabled" usage:"allows to enable and disable webhook sender" default:"true" example:"true / false"`
			Quantity int32 `json:"quantity" yaml:"quantity" usage:"allows you to specify the number of webhooks to send at once" default:"500" example:"500" validate:"gte=1"`
		}
		Cleanup struct {
			Enabled bool          `json:"enabled" yaml:"enabled" usage:"allows to enable and disable webhook cleanup worker" default:"true" example:"true / false"`
			Cron    string        `json:"cron" yaml:"cron" usage:"allows to set custom cron rule for cleanup old competed webhooks" default:"0 1 * * *" example:"0 1 * * *"`
			MaxAge  time.Duration `json:"max_age" yaml:"max_age" usage:"allows to set max age for completed webhooks. by default it 720h (30 days)" default:"720h" example:"720h"`
		}
		RemoveAfterSent bool `yaml:"remove_after_sent" json:"remove_after_sent" usage:"allows to remove a webhook after it has been successfully sent" default:"false" example:"true / false"`
	}
	Interceptors struct {
		DisableCheckingSign bool `yaml:"disable_checking_sign" json:"disable_checking_sign" usage:"allows to disable checking sign of request" default:"false" example:"true / false"`
	}
	Transfers struct {
		Enabled bool `yaml:"enabled" json:"enabled" usage:"allows to enable transfers service" default:"true" example:"true / false"`
	}
	UseCacheForWallets bool          `yaml:"use_cache_for_wallets" json:"use_cache_for_wallets" usage:"allows to use cache for wallets. this option is experimental" default:"true" example:"true / false"`
	MerchantAdmin      MerchantAdmin `yaml:"merchant_admin"`
	Updater            Updater       `yaml:"updater"`
}

func (c Config) IsEnabledSeedEncryption() bool { return true }

type Watcher struct {
	ClientSecret          string                   `yaml:"client_secret"`
	GrpcReconnectionDelay time.Duration            `yaml:"grpc_reconnection_delay" default:"1s" example:"1s"`
	Enabled               bool                     `yaml:"enabled" default:"false"`
	Connect               connectrpc_client.Config `yaml:"connect" default:"test"`
}

func (w *Watcher) Validate() error {
	if !w.Enabled {
		return nil
	}

	if w.Connect.Addr == "" {
		return errors.New("watcher service address is required")
	}

	return nil
}

type ResourceManager struct {
	Enabled              bool                     `yaml:"enabled" default:"false"`
	Connect              connectrpc_client.Config `yaml:"connect" default:"test"`
	UseActivatorContract bool                     `yaml:"use_activator_contract" default:"true"`
	DelegationDuration   time.Duration            `yaml:"delegation_duration" default:"15s" example:"60s/1m/1h" validate:"gte=15s"`
}

func (o *ResourceManager) Validate() error {
	if !o.Enabled {
		return nil
	}

	if o.Connect.Addr == "" {
		return errors.New("resource manager service address is required")
	}

	return nil
}

func (c *Config) SetDefaults() {
	c.ExplorerProxy.Addr = "https://explorer-proxy.dv.net"
	c.ExplorerProxy.Name = "explorer-proxy-client"

	c.ResourceManager.Enabled = true
	c.ResourceManager.Connect.Addr = "https://delegate.dv.net"
	c.ResourceManager.Connect.Name = "orders-client"
	c.ResourceManager.UseActivatorContract = true
	c.ResourceManager.DelegationDuration = 15 * time.Second

	c.Blockchain.Ethereum.Enabled = true
	c.Blockchain.Ethereum.Node.Address = "https://node-eth.dv.net"

	c.Blockchain.BinanceSmartChain.Enabled = true
	c.Blockchain.BinanceSmartChain.Node.Address = "https://node-bsc.dv.net"

	c.Blockchain.Polygon.Enabled = true
	c.Blockchain.Polygon.Node.Address = "https://node-polygon.dv.net"

	c.Blockchain.Arbitrum.Enabled = true
	c.Blockchain.Arbitrum.Node.Address = "https://node-arbitrum.dv.net"

	// c.Blockchain.Optimism.Enabled = true
	// c.Blockchain.Optimism.Node.Address = "https://node-optimism.dv.net"

	// c.Blockchain.Linea.Enabled = true
	// c.Blockchain.Linea.Node.Address = "https://node-linea.dv.net"

	c.Blockchain.Tron.Enabled = true
	c.Blockchain.Tron.Node.GrpcAddress = "node-tron-grpc.dv.net:443"
	c.Blockchain.Tron.Node.UseTLS = true

	c.Blockchain.Bitcoin.Enabled = true
	c.Blockchain.Bitcoin.Node.Address = "node-btc.dv.net:443"
	c.Blockchain.Bitcoin.Node.Login = "rpc"
	c.Blockchain.Bitcoin.Node.Secret = "qh1lFWjT4UPlY0kN"
	c.Blockchain.Bitcoin.Node.UseTLS = true

	c.Blockchain.BitcoinCash.Enabled = true
	c.Blockchain.BitcoinCash.Node.Address = "node-bch.dv.net"
	c.Blockchain.BitcoinCash.Node.Login = "rpc"
	c.Blockchain.BitcoinCash.Node.Secret = "qh1lFWjT4UPlY0kN"
	c.Blockchain.BitcoinCash.Node.UseTLS = true

	c.Blockchain.Litecoin.Enabled = true
	c.Blockchain.Litecoin.Node.Address = "node-ltc.dv.net"
	c.Blockchain.Litecoin.Node.Login = "rpc"
	c.Blockchain.Litecoin.Node.Secret = "qh1lFWjT4UPlY0kN"
	c.Blockchain.Litecoin.Node.UseTLS = true

	c.Blockchain.Dogecoin.Enabled = true
	c.Blockchain.Dogecoin.Node.Address = "node-doge.dv.net:443"
	c.Blockchain.Dogecoin.Node.Login = "rpc"
	c.Blockchain.Dogecoin.Node.Secret = "qh1lFWjT4UPlY0kN"
	c.Blockchain.Dogecoin.Node.UseTLS = true
}
