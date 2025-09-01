package responses

import "time"

type baseResponse struct {
	Message string `json:"message"`
}

type GetIPResponse struct {
	baseResponse
	Data struct {
		IP string `json:"ip"`
	} `json:"data"`
}

type RegisterResponse struct {
	baseResponse
	Data struct {
		Backend struct {
			ID                 string    `json:"id"`
			BackendClientID    string    `json:"backend_client_id"`
			IsEmailEnabled     bool      `json:"is_email_enabled"`
			IP                 string    `json:"ip"`
			ProcessingID       string    `json:"processing_id"`
			Domain             string    `json:"domain"`
			NotificationsCount int       `json:"notifications_count"`
			SecretKey          string    `json:"secret_key"`
			CreatedAt          time.Time `json:"created_at"`
			UpdatedAt          time.Time `json:"updated_at"`
		} `json:"backend"`
		Processing struct {
			ID            string    `json:"id"`
			ExternalID    string    `json:"external_id"`
			IP            string    `json:"ip"`
			SecretKey     string    `json:"secret_key"`
			BackendsCount int       `json:"backends_count"`
			CreatedAt     time.Time `json:"created_at"`
			UpdatedAt     time.Time `json:"updated_at"`
		} `json:"processing"`
	} `json:"data"`
}

type ErrorResponse struct {
	baseResponse
	Errors map[string][]string `json:"errors"`
}
