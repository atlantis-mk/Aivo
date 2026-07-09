package app

import "aivo/core/domain"

func (m *ProviderAuthManager) status(providerID string) domain.ProviderAuthStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	flow := m.flows[providerID]
	if flow == nil {
		return domain.ProviderAuthStatus{ProviderID: providerID, Status: "idle"}
	}
	return domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
}

func (m *ProviderAuthManager) cancel(providerID string) domain.ProviderAuthStatus {
	m.mu.Lock()
	flow := m.flows[providerID]
	if flow == nil {
		m.mu.Unlock()
		return domain.ProviderAuthStatus{ProviderID: providerID, Status: "idle"}
	}
	flow.Status = "cancelled"
	status := domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
	return status
}

func (m *ProviderAuthManager) fail(providerID string, message string) {
	m.mu.Lock()
	var status domain.ProviderAuthStatus
	if flow := m.flows[providerID]; flow != nil {
		flow.Status = "failed"
		flow.Error = message
		status = m.statusFromFlow(flow)
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
}

func (m *ProviderAuthManager) succeed(providerID string, accountID string) {
	m.mu.Lock()
	var status domain.ProviderAuthStatus
	if flow := m.flows[providerID]; flow != nil {
		flow.Status = "success"
		flow.AccountID = accountID
		status = m.statusFromFlow(flow)
	}
	m.mu.Unlock()
	m.emitProviderAuthUpdated(status)
}

func (m *ProviderAuthManager) statusFromFlow(flow *providerAuthFlow) domain.ProviderAuthStatus {
	if flow == nil {
		return domain.ProviderAuthStatus{}
	}
	return domain.ProviderAuthStatus{
		ProviderID:   flow.ProviderID,
		Method:       flow.Method,
		Status:       flow.Status,
		Error:        flow.Error,
		AccountID:    flow.AccountID,
		Instructions: flow.Instructions,
		UserCode:     flow.UserCode,
	}
}

func (m *ProviderAuthManager) emitProviderAuthUpdated(status domain.ProviderAuthStatus) {
	if status.ProviderID == "" || m.service == nil || m.service.onProviderAuthUpdated == nil {
		return
	}
	m.service.onProviderAuthUpdated(status)
}

func (f *providerAuthFlow) startResult() domain.ProviderAuthStartResult {
	return domain.ProviderAuthStartResult{
		ProviderID:   f.ProviderID,
		Method:       f.Method,
		Status:       f.Status,
		URL:          f.URL,
		Instructions: f.Instructions,
		UserCode:     f.UserCode,
		ExpiresAt:    domain.NowString(f.ExpiresAt),
	}
}
