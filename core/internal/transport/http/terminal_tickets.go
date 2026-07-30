package http

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type terminalTicket struct {
	Token         string
	WorkspaceRoot string
	TerminalID    string
	SessionID     string
	ExpiresAt     time.Time
}

func (s *terminalTicketStore) createAgent(workspaceRoot, sessionID, processRef string) (terminalTicket, error) {
	ticket, err := s.create(workspaceRoot, processRef)
	if err != nil {
		return terminalTicket{}, err
	}
	ticket.SessionID = sessionID
	s.mu.Lock()
	s.tickets[ticket.Token] = ticket
	s.mu.Unlock()
	return ticket, nil
}

func (s *terminalTicketStore) consumeAgent(token, workspaceRoot, sessionID, processRef string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.tickets[token]
	if !ok {
		return false
	}
	delete(s.tickets, token)
	return time.Now().Before(ticket.ExpiresAt) && ticket.WorkspaceRoot == workspaceRoot && ticket.SessionID == sessionID && ticket.TerminalID == processRef
}

type terminalTicketStore struct {
	mu      sync.Mutex
	tickets map[string]terminalTicket
}

func newTerminalTicketStore() *terminalTicketStore {
	return &terminalTicketStore{tickets: map[string]terminalTicket{}}
}

func (s *terminalTicketStore) create(workspaceRoot string, terminalID string) (terminalTicket, error) {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err != nil {
		return terminalTicket{}, err
	}
	ticket := terminalTicket{Token: hex.EncodeToString(raw), WorkspaceRoot: workspaceRoot, TerminalID: terminalID, ExpiresAt: time.Now().Add(60 * time.Second)}
	s.mu.Lock()
	s.tickets[ticket.Token] = ticket
	s.mu.Unlock()
	return ticket, nil
}

func (s *terminalTicketStore) consume(token string, workspaceRoot string, terminalID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ticket, ok := s.tickets[token]
	if !ok {
		return false
	}
	delete(s.tickets, token)
	return time.Now().Before(ticket.ExpiresAt) && ticket.WorkspaceRoot == workspaceRoot && ticket.TerminalID == terminalID
}
