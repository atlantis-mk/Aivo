package app

import (
	"context"
	"strings"

	"aivo/core/domain"
)

func (s *Service) saveProviderAuth(ctx context.Context, auth domain.ProviderAuthRecord) error {
	prepared, err := s.prepareProviderAuthSecrets(ctx, auth)
	if err != nil {
		return err
	}
	return s.store.SaveProviderAuth(ctx, prepared)
}

func (s *Service) prepareProviderAuthSecrets(ctx context.Context, auth domain.ProviderAuthRecord) (domain.ProviderAuthRecord, error) {
	if s.secrets == nil {
		s.secrets = NewMemorySecretStore()
	}
	if strings.TrimSpace(auth.APIKey) != "" {
		ref := firstNonEmpty(auth.APIKeyRef, providerSecretRef(auth, "api-key"))
		if err := s.secrets.Put(ctx, ref, auth.APIKey); err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.APIKeyRef = ref
		auth.APIKey = ""
	}
	if strings.TrimSpace(auth.AccessToken) != "" {
		ref := firstNonEmpty(auth.AccessTokenRef, providerSecretRef(auth, "access-token"))
		if err := s.secrets.Put(ctx, ref, auth.AccessToken); err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.AccessTokenRef = ref
		auth.AccessToken = ""
	}
	if strings.TrimSpace(auth.RefreshToken) != "" {
		ref := firstNonEmpty(auth.RefreshTokenRef, providerSecretRef(auth, "refresh-token"))
		if err := s.secrets.Put(ctx, ref, auth.RefreshToken); err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.RefreshTokenRef = ref
		auth.RefreshToken = ""
	}
	return auth, nil
}

func (s *Service) resolveProviderAuthSecrets(ctx context.Context, auth domain.ProviderAuthRecord) (domain.ProviderAuthRecord, error) {
	if s.secrets == nil {
		s.secrets = NewMemorySecretStore()
	}
	if strings.TrimSpace(auth.APIKey) == "" && strings.TrimSpace(auth.APIKeyRef) != "" {
		value, err := s.secrets.Get(ctx, auth.APIKeyRef)
		if err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.APIKey = value
	}
	if strings.TrimSpace(auth.AccessToken) == "" && strings.TrimSpace(auth.AccessTokenRef) != "" {
		value, err := s.secrets.Get(ctx, auth.AccessTokenRef)
		if err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.AccessToken = value
	}
	if strings.TrimSpace(auth.RefreshToken) == "" && strings.TrimSpace(auth.RefreshTokenRef) != "" {
		value, err := s.secrets.Get(ctx, auth.RefreshTokenRef)
		if err != nil {
			return domain.ProviderAuthRecord{}, err
		}
		auth.RefreshToken = value
	}
	return auth, nil
}

func (s *Service) deleteProviderAuthSecrets(ctx context.Context, auth *domain.ProviderAuthRecord) error {
	if auth == nil || s.secrets == nil {
		return nil
	}
	for _, ref := range []string{auth.APIKeyRef, auth.AccessTokenRef, auth.RefreshTokenRef} {
		if strings.TrimSpace(ref) == "" {
			continue
		}
		if err := s.secrets.Delete(ctx, ref); err != nil {
			return err
		}
	}
	return nil
}

func providerSecretRef(auth domain.ProviderAuthRecord, kind string) string {
	account := strings.TrimSpace(auth.AccountID)
	if account == "" {
		account = strings.TrimSpace(auth.ID)
	}
	if account == "" {
		account = "default"
	}
	return "provider-auth/" + cleanSecretRefPart(auth.ProviderID) + "/" + cleanSecretRefPart(auth.Method) + "/" + cleanSecretRefPart(account) + "/" + kind
}

func cleanSecretRefPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "default"
	}
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return "default"
	}
	return out
}
