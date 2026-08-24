package http

import (
	"testing"
	"time"
)

func TestAgentTerminalTicketIsSingleUseAndOwnerScoped(t *testing.T) {
	store := newTerminalTicketStore()
	ticket, err := store.createAgent("/workspace", "session-1", "agent-pty:1")
	if err != nil {
		t.Fatal(err)
	}
	if store.consumeAgent(ticket.Token, "/workspace", "other", "agent-pty:1") {
		t.Fatal("cross-session ticket was accepted")
	}
	if store.consumeAgent(ticket.Token, "/workspace", "session-1", "agent-pty:1") {
		t.Fatal("failed attempt did not consume ticket")
	}
	ticket, err = store.createAgent("/workspace", "session-1", "agent-pty:1")
	if err != nil {
		t.Fatal(err)
	}
	if !store.consumeAgent(ticket.Token, "/workspace", "session-1", "agent-pty:1") {
		t.Fatal("valid ticket was rejected")
	}
	if store.consumeAgent(ticket.Token, "/workspace", "session-1", "agent-pty:1") {
		t.Fatal("ticket was reused")
	}
}

func TestAgentTerminalTicketExpires(t *testing.T) {
	store := newTerminalTicketStore()
	ticket, err := store.createAgent("/workspace", "session-1", "agent-pty:1")
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	value := store.tickets[ticket.Token]
	value.ExpiresAt = time.Now().Add(-time.Second)
	store.tickets[ticket.Token] = value
	store.mu.Unlock()
	if store.consumeAgent(ticket.Token, "/workspace", "session-1", "agent-pty:1") {
		t.Fatal("expired ticket was accepted")
	}
}
