package clients

import "github.com/dv-net/dv-processing/internal/models"

type CreateClientResult struct {
	AdminSecret string
	Client      *models.Client
}

type CreateClientDTO struct {
	CallbackURL    string
	BackendVersion string
	BackendAddress *string
	BackendDomain  *string
}
