package app

import (
	"context"
	"errors"
	"strings"
	"sync"

	"aivo/core/domain"
)

type HostCredentialBroker struct {
	mu       sync.Mutex
	store    SecretStore
	bindings map[string]string
	leases   map[string]map[string]bool
}

func NewHostCredentialBroker(store SecretStore) *HostCredentialBroker {
	return &HostCredentialBroker{store: store, bindings: map[string]string{}, leases: map[string]map[string]bool{}}
}

func (b *HostCredentialBroker) SetStore(store SecretStore) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.store = store
}

func (b *HostCredentialBroker) Bind(extensionID, slot, secretRef string) error {
	extensionID, slot, secretRef = strings.TrimSpace(extensionID), strings.TrimSpace(slot), strings.TrimSpace(secretRef)
	if extensionID == "" || slot == "" || secretRef == "" {
		return errors.New("extensionId, credential slot, and secretRef are required")
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.bindings[extensionID+"\x00"+slot] = secretRef
	return nil
}

func (b *HostCredentialBroker) Request(ctx context.Context, extensionID string, declaredSlots []string, slot, operationID string) (string, error) {
	extensionID, slot, operationID = strings.TrimSpace(extensionID), strings.TrimSpace(slot), strings.TrimSpace(operationID)
	if operationID == "" {
		return "", errors.New("credential requests require an operation owner")
	}
	declared := false
	for _, name := range declaredSlots {
		if strings.TrimSpace(name) == slot {
			declared = true
		}
	}
	if !declared {
		return "", errors.New("credential slot is not declared by the extension")
	}
	b.mu.Lock()
	ref := b.bindings[extensionID+"\x00"+slot]
	store := b.store
	if ref != "" {
		if b.leases[operationID] == nil {
			b.leases[operationID] = map[string]bool{}
		}
		b.leases[operationID][extensionID+"\x00"+slot] = true
	}
	b.mu.Unlock()
	if ref == "" || store == nil {
		return "", errors.New("credential slot is not bound")
	}
	value, err := store.Get(ctx, ref)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", errors.New("bound credential is unavailable")
	}
	return value, nil
}

func (b *HostCredentialBroker) Release(operationID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.leases, strings.TrimSpace(operationID))
}

func (s *Service) BindExtensionCredential(_ context.Context, input domain.BindExtensionCredentialInput) (domain.ExtensionCredentialBinding, error) {
	if s.extensionSupervisor == nil {
		return domain.ExtensionCredentialBinding{}, errors.New("extension supervisor is unavailable")
	}
	status, err := s.extensionSupervisor.Status(input.ID)
	if err != nil {
		return domain.ExtensionCredentialBinding{}, err
	}
	s.extensionSupervisor.mu.Lock()
	item := s.extensionSupervisor.items[input.ID]
	declared := false
	if item != nil {
		for _, slot := range item.loaded.Manifest.Requirements.Credentials {
			if slot == input.Slot {
				declared = true
			}
		}
	}
	s.extensionSupervisor.mu.Unlock()
	if !status.Trusted || !declared {
		return domain.ExtensionCredentialBinding{}, errors.New("extension must be trusted and declare the credential slot")
	}
	if s.extensionCredentials == nil {
		s.extensionCredentials = NewHostCredentialBroker(s.secrets)
	}
	if err := s.extensionCredentials.Bind(input.ID, input.Slot, input.SecretRef); err != nil {
		return domain.ExtensionCredentialBinding{}, err
	}
	return domain.ExtensionCredentialBinding{ID: input.ID, Slot: input.Slot, Bound: true}, nil
}
