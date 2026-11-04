package agentic

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/caveman/go-rt/internal/domain"
)

// ConversationMemory manages active multi-turn dialogue state and backtracked audit history.
type ConversationMemory struct {
	mu     sync.Mutex
	active []domain.ConversationTurn
	audit  []domain.ConversationTurn
}

// NewConversationMemory creates an empty multi-turn conversation buffer.
func NewConversationMemory() *ConversationMemory {
	return &ConversationMemory{}
}

// AppendExchange records a completed user/assistant turn in both active and audit buffers.
func (m *ConversationMemory) AppendExchange(prompt, output string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	turn := domain.ConversationTurn{
		Turn:      len(m.active) + 1,
		Role:      "exchange",
		Prompt:    prompt,
		Output:    output,
		Timestamp: time.Now().UTC(),
	}
	m.active = append(m.active, turn)
	m.audit = append(m.audit, turn)
}

// RollbackLastTurn removes the latest turn from active context and marks it as backtracked in audit history.
func (m *ConversationMemory) RollbackLastTurn() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.active) == 0 {
		return false
	}
	m.active = m.active[:len(m.active)-1]
	for i := len(m.audit) - 1; i >= 0; i-- {
		if !m.audit[i].Backtrack {
			m.audit[i].Backtrack = true
			break
		}
	}
	return true
}

// ActiveTurns returns a copy of the non-rolled-back conversation turns.
func (m *ConversationMemory) ActiveTurns() []domain.ConversationTurn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.active)
}

// AuditHistory returns a copy of all turns, including rolled-back attempts.
func (m *ConversationMemory) AuditHistory() []domain.ConversationTurn {
	m.mu.Lock()
	defer m.mu.Unlock()
	return slices.Clone(m.audit)
}

// FormatTranscript renders the active conversation history into a dialogue string.
func (m *ConversationMemory) FormatTranscript() string {
	turns := m.ActiveTurns()
	if len(turns) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, t := range turns {
		sb.WriteString(fmt.Sprintf("Turn %d User: %s\nTurn %d Assistant: %s\n\n",
			t.Turn, t.Prompt, t.Turn, t.Output))
	}
	return strings.TrimSpace(sb.String())
}

// BuildMessages converts active dialogue turns into structured role-based chat messages.
func (m *ConversationMemory) BuildMessages(nextPrompt string) []domain.Message {
	turns := m.ActiveTurns()
	msgs := make([]domain.Message, 0, len(turns)*2+1)
	for _, t := range turns {
		msgs = append(msgs, domain.Message{Role: "user", Content: t.Prompt})
		msgs = append(msgs, domain.Message{Role: "assistant", Content: t.Output})
	}
	msgs = append(msgs, domain.Message{Role: "user", Content: nextPrompt})
	return msgs
}

// BuildContextualPrompt combines prior active turns with the current user message for stateless providers.
func (m *ConversationMemory) BuildContextualPrompt(nextPrompt string) string {
	transcript := m.FormatTranscript()
	if transcript == "" {
		return nextPrompt
	}
	return fmt.Sprintf("Conversation History:\n%s\n\nCurrent User Message:\n%s", transcript, nextPrompt)
}
