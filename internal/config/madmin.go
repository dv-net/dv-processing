package config

type MerchantAdmin struct {
	BaseURL string `yaml:"base_url" json:"base_url" usage:"allows to set merchant admin service endpoint" default:"https://api.dv.net" example:"https://api.dv.net"`
}
