package clients

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	madmin_requests "github.com/dv-net/dv-processing/internal/madmin/requests"
	"github.com/dv-net/dv-processing/internal/models"
	"github.com/dv-net/dv-processing/internal/store/repos"
	"github.com/dv-net/dv-processing/internal/store/storecmn"
	"github.com/dv-net/dv-processing/pkg/utils"
)

// Create creates a new client.
func (s *Service) Create(ctx context.Context, dto CreateClientDTO) (*CreateClientResult, error) {
	client := &models.Client{}

	secretKey, err := utils.SpecialKey(32)
	if err != nil {
		return nil, fmt.Errorf("key random generate: %w", err)
	}

	if !validateCallbackURL(dto.CallbackURL) {
		return nil, fmt.Errorf("invalid callback url: %s", dto.CallbackURL)
	}

	if dto.BackendAddress != nil {
		if !validateExternalIP(*dto.BackendAddress) {
			return nil, fmt.Errorf("invalid backend address: %s", *dto.BackendAddress)
		}
	}

	processingIP, err := getPublicAddress(ctx)
	if err != nil {
		res, err := s.madminSvc.GetIP(ctx)
		if err != nil {
			return nil, err
		}
		processingIP = res.Data.IP
	}

	if !validateExternalIP(processingIP) {
		return nil, fmt.Errorf("invalid processing ip: %s", processingIP)
	}

	var adminSecretKey string
	err = pgx.BeginTxFunc(ctx, s.store.PSQLConn(), pgx.TxOptions{}, func(tx pgx.Tx) error {
		client, err = s.store.Clients(repos.WithTx(tx)).Create(ctx, secretKey, dto.CallbackURL)
		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				return storecmn.ErrAlreadyExists
			}
			return err
		}

		req := &madmin_requests.RegisterRequest{
			BackendClientID:   client.ID.String(),
			BackendVersion:    dto.BackendVersion,
			ProcessingVersion: s.systemSvc.SystemVersion(ctx),
			ProcessingIP:      processingIP,
		}

		if dto.BackendAddress != nil {
			req.BackendIP = *dto.BackendAddress
		}
		if dto.BackendDomain != nil {
			req.BackendDomain = *dto.BackendDomain
		}

		registerResp, err := s.madminSvc.Register(ctx, req)
		if err != nil {
			return err
		}

		if registerResp != nil {
			if err = s.systemSvc.SetDvSecretKey(ctx, registerResp.Data.Processing.SecretKey); err != nil {
				return fmt.Errorf("set secret key: %w", err)
			}

			adminSecretKey = registerResp.Data.Backend.SecretKey
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &CreateClientResult{
		AdminSecret: adminSecretKey,
		Client:      client,
	}, nil
}

// GetAll returns all clients.
func (s *Service) GetAll(ctx context.Context) ([]*models.Client, error) {
	return s.store.Clients().GetAll(ctx)
}

// GetByID returns the client by the ID.
func (s *Service) GetByID(ctx context.Context, clientID uuid.UUID) (*models.Client, error) {
	if clientID == uuid.Nil {
		return nil, storecmn.ErrEmptyID
	}

	data, err := s.store.Clients().GetByID(ctx, clientID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storecmn.ErrNotFound
		}
		return nil, err
	}

	return data, nil
}

// ExistsByID checks if the client exists by the ID.
func (s *Service) ExistsByID(ctx context.Context, clientID uuid.UUID) (bool, error) {
	if clientID == uuid.Nil {
		return false, storecmn.ErrEmptyID
	}

	return s.store.Clients().ExistsByID(ctx, clientID)
}

// ChangeCallbackURL changes the callback URL for the client.
func (s *Service) ChangeCallbackURL(ctx context.Context, clientID uuid.UUID, callbackURL string) error {
	if clientID == uuid.Nil {
		return storecmn.ErrEmptyID
	}

	// validate callback url
	if !validateCallbackURL(callbackURL) {
		return fmt.Errorf("invalid callback url: %s", callbackURL)
	}

	// get client
	exists, err := s.ExistsByID(ctx, clientID)
	if err != nil {
		return err
	}

	if !exists {
		return storecmn.ErrNotFound
	}

	s.store.Cache().Clients().Delete(clientID.String())

	return s.store.Clients().ChangeCallbackURL(ctx, callbackURL, clientID)
}
