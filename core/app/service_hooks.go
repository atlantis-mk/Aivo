package app

import "aivo/core/domain"

func (s *Service) SetAuthSuccessHook(hook func()) {
	s.onAuthSuccess = hook
}

func (s *Service) SetProviderAuthUpdatedHook(hook func(domain.ProviderAuthStatus)) {
	s.onProviderAuthUpdated = hook
}

func (s *Service) SetSessionUpdatedHook(hook func(string, *domain.Session)) {
	s.onSessionUpdated = hook
}

func (s *Service) SetTurnUpdatedHook(hook func(string, domain.Turn)) {
	s.onTurnUpdated = hook
}

func (s *Service) SetAssistantDeltaHook(hook func(sessionID string, turnID string, delta string)) {
	s.onAssistantDelta = hook
}

func (s *Service) SetToolCallUpdatedHook(hook func(sessionID string, turnID string, call domain.ToolCall, created bool)) {
	s.onToolCallUpdated = hook
}

func (s *Service) SetShellOutputHook(hook func(ShellOutputEvent)) {
	s.onShellOutput = hook
}

func (s *Service) SetTodoItemsUpdatedHook(hook func(sessionID string, projectPath string, items []domain.TodoItem)) {
	s.onTodoItemsUpdated = hook
}

func (s *Service) SetPermissionRequestedHook(hook func(domain.PermissionRequest)) {
	s.onPermissionRequested = hook
}

func (s *Service) SetPermissionResolvedHook(hook func(domain.PermissionRequest)) {
	s.onPermissionResolved = hook
}

func (s *Service) SetQuestionRequestedHook(hook func(domain.QuestionRequest)) {
	s.onQuestionRequested = hook
}

func (s *Service) SetQuestionResolvedHook(hook func(domain.QuestionRequest)) {
	s.onQuestionResolved = hook
}

func (s *Service) SetTerminalEventHook(hook func(string, TerminalInfo)) {
	s.onTerminalEvent = hook
}
