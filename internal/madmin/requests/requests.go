package requests

type RegisterRequest struct {
	BackendClientID   string `json:"backend_client_id" validate:"required,uuid4"`
	BackendVersion    string `json:"backend_version" validate:"required"`
	ProcessingVersion string `json:"processing_version" validate:"required"`
	ProcessingID      string `json:"processing_id" validate:"required,uuid4"`
	BackendDomain     string `json:"backend_domain"`
	BackendIP         string `json:"backend_ip" validate:"ip_addr"`
	ProcessingIP      string `json:"processing_ip" validate:"ip_addr"`
}
