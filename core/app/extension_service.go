package app

import (
	"context"
	"errors"
	"strings"

	"aivo/core/domain"
)

func (s *Service) DiscoverExtension(_ context.Context, input domain.DiscoverExtensionInput) (domain.ExtensionStatus, error) {
	if s.extensionSupervisor == nil {
		s.extensionSupervisor = NewExtensionSupervisor()
	}
	return s.extensionSupervisor.Discover(input.Path)
}

func (s *Service) TrustExtension(_ context.Context, input domain.TrustExtensionInput) (domain.ExtensionStatus, error) {
	if s.extensionSupervisor == nil {
		return domain.ExtensionStatus{}, errors.New("extension supervisor is unavailable")
	}
	return s.extensionSupervisor.Trust(input.ID, input.Integrity)
}

func (s *Service) EnableExtension(ctx context.Context, input domain.ExtensionControlInput) (domain.ExtensionStatus, error) {
	if s.extensionSupervisor == nil {
		return domain.ExtensionStatus{}, errors.New("extension supervisor is unavailable")
	}
	return s.extensionSupervisor.Enable(ctx, input.ID)
}

func (s *Service) StopExtension(ctx context.Context, input domain.ExtensionControlInput) (domain.ExtensionStatus, error) {
	if s.extensionSupervisor == nil {
		return domain.ExtensionStatus{}, errors.New("extension supervisor is unavailable")
	}
	return s.extensionSupervisor.Stop(ctx, input.ID)
}

func (s *Service) GetExtensionStatus(_ context.Context, input domain.ExtensionControlInput) (domain.ExtensionStatus, error) {
	if s.extensionSupervisor == nil {
		return domain.ExtensionStatus{}, errors.New("extension supervisor is unavailable")
	}
	return s.extensionSupervisor.Status(input.ID)
}

func (s *Service) ListExtensionContexts(_ context.Context, input domain.ExtensionControlInput) ([]domain.ExtensionContextResource, error) {
	if s.extensionSupervisor == nil || strings.TrimSpace(input.ID) == "" {
		return nil, errors.New("extensionId is required")
	}
	return s.extensionSupervisor.ContextResources(input.ID)
}

func (s *Service) ResolveExtensionView(ctx context.Context, input domain.ResolveExtensionViewInput) (domain.ExtensionViewDescriptor, error) {
	if s.extensionSupervisor == nil || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.ViewID) == "" {
		return domain.ExtensionViewDescriptor{}, errors.New("extensionId and viewId are required")
	}
	return s.extensionSupervisor.ResolveView(ctx, input.ID, input.ViewID)
}

func (s *Service) OpenExtensionView(ctx context.Context, input domain.ExtensionControlInput) error {
	if s.extensionSupervisor == nil || strings.TrimSpace(input.ID) == "" {
		return errors.New("extensionId is required")
	}
	return s.extensionSupervisor.OpenView(ctx, input.ID)
}

func (s *Service) CloseExtensionView(ctx context.Context, input domain.ExtensionControlInput) error {
	if s.extensionSupervisor == nil || strings.TrimSpace(input.ID) == "" {
		return errors.New("extensionId is required")
	}
	return s.extensionSupervisor.CloseView(ctx, input.ID)
}

func (s *Service) InvokeExtensionViewAction(ctx context.Context, input domain.ExtensionViewActionInput) (any, error) {
	if s.extensionSupervisor == nil || strings.TrimSpace(input.ID) == "" || strings.TrimSpace(input.ViewID) == "" || strings.TrimSpace(input.Action) == "" {
		return nil, errors.New("extensionId, viewId, and action are required")
	}
	return s.extensionSupervisor.InvokeViewAction(ctx, input.ID, input.ViewID, input.Action, input.Data)
}
