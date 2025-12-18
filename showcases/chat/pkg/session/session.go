package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Message represents a single chat message
type Message struct {
	ID        string    `json:"id"`        // unique message id
	Role      string    `json:"role"`      // "user" or "assistant"
	Content   string    `json:"content"`   // message content
	Timestamp time.Time `json:"timestamp"` // when the message was sent
	Feedback  string    `json:"feedback"`  // "like", "dislike", or empty
}

// Session represents a chat session with history
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	mu        sync.RWMutex
}

// SessionManager manages multiple chat sessions
type SessionManager struct {
	sessions   map[string]*Session
	sessionDir string
	maxHistory int
	mu         sync.RWMutex
}

// NewSessionManager creates a new session manager
func NewSessionManager(sessionDir string, maxHistory int) *SessionManager {
	// Create sessions directory if it doesn't exist
	os.MkdirAll(sessionDir, 0755)

	sm := &SessionManager{
		sessions:   make(map[string]*Session),
		sessionDir: sessionDir,
		maxHistory: maxHistory,
	}

	// Don't load all sessions at startup anymore
	// Sessions will be loaded on-demand to improve startup performance
	// sm.loadSessions()

	return sm
}

// GetSessionDir returns the session directory
func (sm *SessionManager) GetSessionDir() string {
	return sm.sessionDir
}

// GetMaxHistory returns the maximum history length
func (sm *SessionManager) GetMaxHistory() int {
	return sm.maxHistory
}

// CreateSession creates a new chat session
func (sm *SessionManager) CreateSession() *Session {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session := &Session{
		ID:        uuid.New().String(),
		Messages:  make([]Message, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	sm.sessions[session.ID] = session
	// Don't save new sessions until they have messages

	return session
}

// GetSession retrieves a session by ID (lazy loads from disk if not in memory)
func (sm *SessionManager) GetSession(id string) (*Session, error) {
	// First check if session is already loaded
	sm.mu.RLock()
	session, exists := sm.sessions[id]
	sm.mu.RUnlock()

	if exists {
		return session, nil
	}

	// Try to load from disk
	return sm.loadSessionFromDisk(id)
}

// ListSessions returns all active sessions
func (sm *SessionManager) ListSessions() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sessions := make([]*Session, 0, len(sm.sessions))
	for _, session := range sm.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// DeleteSession removes a session
func (sm *SessionManager) DeleteSession(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, id)

	// Delete from disk
	filePath := filepath.Join(sm.sessionDir, fmt.Sprintf("%s.json", id))
	return os.Remove(filePath)
}

// AddMessage adds a message to a session
func (sm *SessionManager) AddMessage(sessionID, role, content string) (string, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return "", err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	msgID := uuid.New().String()
	message := Message{
		ID:        msgID,
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}

	session.Messages = append(session.Messages, message)
	session.UpdatedAt = time.Now()

	// Trim history if too long
	if sm.maxHistory > 0 && len(session.Messages) > sm.maxHistory {
		session.Messages = session.Messages[len(session.Messages)-sm.maxHistory:]
	}

	// Save to disk
	sm.saveSession(session)

	return msgID, nil
}

// UpdateMessageFeedback updates the feedback for a specific message
func (sm *SessionManager) UpdateMessageFeedback(sessionID, messageID, feedback string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	found := false
	for i := range session.Messages {
		if session.Messages[i].ID == messageID {
			session.Messages[i].Feedback = feedback
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("message not found: %s", messageID)
	}

	session.UpdatedAt = time.Now()
	
	// Save to disk
	return sm.saveSession(session)
}

// GetMessages retrieves all messages from a session
func (sm *SessionManager) GetMessages(sessionID string) ([]Message, error) {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	session.mu.RLock()
	defer session.mu.RUnlock()

	// Return a copy to prevent external modification
	messages := make([]Message, len(session.Messages))
	copy(messages, session.Messages)

	return messages, nil
}

// saveSession saves a session to disk
func (sm *SessionManager) saveSession(session *Session) error {
	// Only save sessions that have messages
	if len(session.Messages) == 0 {
		// If the session has no messages, don't save it to disk
		// If it exists on disk from before, delete it
		filePath := filepath.Join(sm.sessionDir, fmt.Sprintf("%s.json", session.ID))
		if _, err := os.Stat(filePath); err == nil {
			// File exists, delete it
			os.Remove(filePath)
		}
		return nil
	}

	filePath := filepath.Join(sm.sessionDir, fmt.Sprintf("%s.json", session.ID))

	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// loadSessions loads all sessions from disk
func (sm *SessionManager) loadSessions() {
	files, err := os.ReadDir(sm.sessionDir)
	if err != nil {
		return // Directory might not exist yet
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(sm.sessionDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var session Session
		err = json.Unmarshal(data, &session)
		if err != nil {
			continue
		}

		// Only load sessions that have messages
		if len(session.Messages) > 0 {
			sm.sessions[session.ID] = &session
		}
	}
}

// loadSessionFromDisk loads a specific session from disk
func (sm *SessionManager) loadSessionFromDisk(sessionID string) (*Session, error) {
	filePath := filepath.Join(sm.sessionDir, sessionID+".json")

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("session file not found: %s", sessionID)
	}

	var session Session
	err = json.Unmarshal(data, &session)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal session: %v", err)
	}

	// Store in memory for future access
	sm.mu.Lock()
	sm.sessions[sessionID] = &session
	sm.mu.Unlock()

	return &session, nil
}

// ClearHistory clears all messages in a session
func (sm *SessionManager) ClearHistory(sessionID string) error {
	session, err := sm.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	session.Messages = make([]Message, 0)
	session.UpdatedAt = time.Now()

	// Delete the file from disk since the session has no messages
	filePath := filepath.Join(sm.sessionDir, fmt.Sprintf("%s.json", session.ID))
	os.Remove(filePath)

	return nil
}
