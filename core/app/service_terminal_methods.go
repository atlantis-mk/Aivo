package app

import "context"

func (s *Service) ListTerminals(ctx context.Context, workspaceRoot string) ([]TerminalInfo, error) {
	return s.terminals.List(ctx, workspaceRoot)
}

func (s *Service) CreateTerminal(ctx context.Context, input TerminalCreateInput) (TerminalInfo, error) {
	return s.terminals.Create(ctx, input)
}

func (s *Service) GetTerminal(ctx context.Context, workspaceRoot string, terminalID string) (TerminalInfo, error) {
	return s.terminals.Get(ctx, workspaceRoot, terminalID)
}

func (s *Service) UpdateTerminal(ctx context.Context, input TerminalUpdateInput) (TerminalInfo, error) {
	return s.terminals.Update(ctx, input)
}

func (s *Service) RemoveTerminal(ctx context.Context, workspaceRoot string, terminalID string) error {
	return s.terminals.Remove(ctx, workspaceRoot, terminalID)
}

func (s *Service) AttachTerminal(ctx context.Context, input TerminalAttachInput) (TerminalAttachment, error) {
	return s.terminals.Attach(ctx, input)
}

func (s *Service) PollShellProcess(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Poll(id)
}

func (s *Service) WaitShellProcess(ctx context.Context, id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Wait(ctx, id)
}

func (s *Service) KillShellProcess(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.Kill(id)
}

func (s *Service) ReadShellProcessOutput(id string) (ShellProcessInfo, error) {
	return defaultShellProcessRegistry.ReadOutput(id)
}
