package main

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	mcpclient "github.com/smallnest/goskills/mcp"
	"github.com/smallnest/langgraphgo/adapter/mcp"
	adaptergoskills "github.com/smallnest/langgraphgo/adapter/goskills"
	"github.com/smallnest/goskills"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
	"github.com/tmc/langchaingo/tools"
)

// Config holds application configuration
type Config struct {
	ChatTitle     string
	OpenAIAPIKey  string
	OpenAIModel   string
	OpenAIBaseURL string
}

// loadEnv loads environment variables from .env file if it exists
func loadEnv() {
	if _, err := os.Stat(".env"); err == nil {
		content, err := os.ReadFile(".env")
		if err != nil {
			log.Printf("Error reading .env file: %v", err)
			return
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				os.Setenv(key, value)
			}
		}
	}
}

// getConfig returns application configuration
func getConfig() Config {
	return Config{
		ChatTitle:     getEnvOrDefault("CHAT_TITLE", "LangGraphGo 聊天"),
		OpenAIAPIKey:  os.Getenv("OPENAI_API_KEY"),
		OpenAIModel:   getEnvOrDefault("OPENAI_MODEL", "gpt-4o-mini"),
		OpenAIBaseURL: os.Getenv("OPENAI_BASE_URL"),
	}
}

// getEnvOrDefault returns environment variable or default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// SkillInfo stores basic info about a skill
type SkillInfo struct {
	Name        string
	Description string
	Package     *goskills.SkillPackage
	Tools       []tools.Tool // Cached tools for the skill
	Loaded      bool         // Whether tools have been loaded
}

// ChatAgent interface defines the contract for chat agents
type ChatAgent interface {
	Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error)
}

// SimpleChatAgent manages conversation history for a session
type SimpleChatAgent struct {
	llm           llms.Model
	messages      []llms.MessageContent
	mu            sync.RWMutex
	mcpClient     *mcpclient.Client
	mcpTools      []tools.Tool
	skills        []SkillInfo
	selectedSkill string // Currently selected skill name
	toolsEnabled  bool
	toolsLoading  bool // true when tools are being loaded asynchronously
	toolsLoaded   bool // true when tools have finished loading
}

// NewSimpleChatAgent creates a simple chat agent
func NewSimpleChatAgent(llm llms.Model, config Config) *SimpleChatAgent {
	// Add system message
	systemMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeSystem,
		Parts: []llms.ContentPart{llms.TextPart("You are a helpful AI assistant. Be concise and friendly.")},
	}

	agent := &SimpleChatAgent{
		llm:      llm,
		messages: []llms.MessageContent{systemMsg},
	}

	return agent
}

// InitializeToolsAsync asynchronously loads Skills and MCP tools in the background
// This prevents blocking server startup while tools are being loaded
func (a *SimpleChatAgent) InitializeToolsAsync() {
	// Mark as loading
	a.mu.Lock()
	a.toolsLoading = true
	a.toolsLoaded = false
	a.mu.Unlock()

	go func() {
		log.Println("Starting background tools initialization...")

		// Load Skills
		skillsDir := os.Getenv("SKILLS_DIR")
		if skillsDir == "" {
			skillsDir = "../../testdata/skills"
		}

		if _, err := os.Stat(skillsDir); err == nil {
			packages, err := goskills.ParseSkillPackages(skillsDir)
			if err != nil {
				log.Printf("Failed to parse skills packages: %v", err)
			} else {
				a.mu.Lock()
				for _, skill := range packages {
					// Store skill info without converting to tools yet
					a.skills = append(a.skills, SkillInfo{
						Name:        skill.Meta.Name,
						Description: skill.Meta.Description,
						Package:     skill,
						Loaded:      false,
					})
				}
				a.toolsEnabled = true
				a.mu.Unlock()
				log.Printf("Loaded %d skills info", len(packages))
			}
		} else {
			log.Printf("Skills directory not found at %s", skillsDir)
		}

		// Load MCP
		mcpConfigPath := os.Getenv("MCP_CONFIG_PATH")
		if mcpConfigPath == "" {
			mcpConfigPath = "../../testdata/mcp/mcp.json"
		}

		// Safely initialize MCP with error recovery
		if err := a.initializeMCP(mcpConfigPath); err != nil {
			log.Printf("MCP initialization failed (continuing without MCP): %v", err)
		}

		// Mark as loaded
		a.mu.Lock()
		a.toolsLoading = false
		a.toolsLoaded = true
		a.mu.Unlock()

		log.Println("Background tools initialization complete")
	}()
}

// initializeMCP safely initializes MCP client with error recovery
func (a *SimpleChatAgent) initializeMCP(mcpConfigPath string) (err error) {
	// Add panic recovery to prevent crashes from MCP initialization
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP initialization: %v", r)
			log.Printf("Recovered from MCP initialization panic: %v", r)
		}
	}()

	// Use a longer timeout for initialization as npx downloads may be slow
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Load MCP config
	config, err := mcpclient.LoadConfig(mcpConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load MCP config: %w", err)
	}

	// Create MCP client with error handling
	client, err := mcpclient.NewClient(ctx, config)
	if err != nil {
		return fmt.Errorf("failed to create MCP client: %w", err)
	}

	// Get tools from MCP with timeout
	toolsCtx, toolsCancel := context.WithTimeout(ctx, 30*time.Second)
	defer toolsCancel()

	tools, err := mcp.MCPToTools(toolsCtx, client)
	if err != nil {
		// Close client if tool loading fails
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client after error: %v", closeErr)
		}
		return fmt.Errorf("failed to get MCP tools: %w", err)
	}

	if len(tools) == 0 {
		log.Printf("No MCP tools found, closing client")
		if closeErr := a.closeMCPClient(client); closeErr != nil {
			log.Printf("Failed to close MCP client: %v", closeErr)
		}
		return nil
	}

	// Successfully initialized
	a.mu.Lock()
	a.mcpClient = client
	a.mcpTools = tools
	a.toolsEnabled = true
	a.mu.Unlock()
	log.Printf("Successfully loaded %d MCP tools", len(tools))

	return nil
}

// closeMCPClient safely closes an MCP client with panic recovery and timeout
func (a *SimpleChatAgent) closeMCPClient(client *mcpclient.Client) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during MCP client close: %v", r)
			log.Printf("Recovered from MCP client close panic: %v", r)
		}
	}()

	if client == nil {
		return nil
	}

	// Use a goroutine with timeout to prevent hanging on close
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("panic in close goroutine: %v", r)
			}
		}()
		done <- client.Close()
	}()

	// Wait for close with timeout
	select {
	case closeErr := <-done:
		if closeErr != nil {
			return fmt.Errorf("failed to close MCP client: %w", closeErr)
		}
		return nil
	case <-time.After(5 * time.Second):
		log.Printf("Warning: MCP client close timed out after 5 seconds")
		return fmt.Errorf("MCP client close timed out")
	}
}

// Close releases resources held by the agent
func (a *SimpleChatAgent) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	log.Printf("Closing agent and cleaning up resources...")

	if a.mcpClient != nil {
		log.Printf("Closing MCP client...")
		if err := a.closeMCPClient(a.mcpClient); err != nil {
			// Log error but don't return - we want to continue cleanup
			log.Printf("Error closing MCP client (continuing cleanup): %v", err)
		}
		a.mcpClient = nil
		a.mcpTools = nil
		log.Printf("MCP client closed and cleared")
	}

	return nil
}

// getToolsInfo returns a formatted string of available tools
func (a *SimpleChatAgent) getToolsInfo() string {
	if len(a.mcpTools) == 0 {
		return "No tools available."
	}

	var info strings.Builder
	for _, tool := range a.mcpTools {
		info.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}
	return info.String()
}

// GetAvailableTools returns the list of available skills and MCP tools
func (a *SimpleChatAgent) GetAvailableTools() []map[string]string {
	var tools []map[string]string

	// Add MCP tools
	for _, tool := range a.mcpTools {
		tools = append(tools, map[string]string{
			"name":        tool.Name(),
			"description": tool.Description(),
			"type":        "mcp",
		})
	}

	// Add skills (not loaded as tools yet)
	for _, skill := range a.skills {
		tools = append(tools, map[string]string{
			"name":        skill.Name,
			"description": skill.Description,
			"type":        "skill",
		})
	}

	return tools
}

// Chat sends a message and returns response
func (a *SimpleChatAgent) Chat(ctx context.Context, message string, enableSkills bool, enableMCP bool) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add user message
	userMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeHuman,
		Parts: []llms.ContentPart{llms.TextPart(message)},
	}
	a.messages = append(a.messages, userMsg)

	toolUsed := false
	var toolResult string
	var toolName string

	if a.toolsEnabled {
		// Stage 1: Select skill if needed (only if user enables Skills)
		if enableSkills && len(a.skills) > 0 {
			selectedSkill, err := a.selectSkillForTask(ctx, message)
			if err != nil {
				log.Printf("Skill selection error: %v", err)
			} else if selectedSkill != "" {
				// Load tools for the selected skill
				skillTools, err := a.loadSkillTools(selectedSkill)
				if err != nil {
					log.Printf("Failed to load skill tools: %v", err)
				} else {
					a.selectedSkill = selectedSkill

					// Stage 2: Select specific tool from the skill
					tool, args, err := a.selectToolForTask(ctx, message, skillTools)
					if err != nil {
						log.Printf("Tool selection error: %v", err)
					} else if tool != nil {
						// Convert args to JSON string
						argsJSON, _ := json.Marshal(args)
						argsStr := string(argsJSON)
						if argsStr == "null" {
							argsStr = "{}"
						}

						// Call the tool
						result, err := (*tool).Call(ctx, argsStr)
						if err != nil {
							log.Printf("Tool %s call failed: %v", (*tool).Name(), err)
						} else {
							toolUsed = true
							toolResult = result
							toolName = (*tool).Name()
							log.Printf("Successfully used tool '%s' from skill '%s'", (*tool).Name(), selectedSkill)
						}
					}
				}
			}
		}

		// If no skill was selected, try MCP tools (only if user enables MCP)
		if !toolUsed && enableMCP && len(a.mcpTools) > 0 {
			tool, args, err := a.selectToolForTask(ctx, message, a.mcpTools)
			if err != nil {
				log.Printf("MCP tool selection error: %v", err)
			} else if tool != nil {
				// Convert args to JSON string
				argsJSON, _ := json.Marshal(args)
				argsStr := string(argsJSON)
				if argsStr == "null" {
					argsStr = "{}"
				}

				// Call the tool
				result, err := (*tool).Call(ctx, argsStr)
				if err != nil {
					log.Printf("MCP tool %s call failed: %v", (*tool).Name(), err)
				} else {
					toolUsed = true
					toolResult = result
					toolName = (*tool).Name()
					log.Printf("Successfully used MCP tool '%s'", (*tool).Name())
				}
			}
		}
	}

	// Add tool result to conversation if a tool was used
	if toolUsed && toolResult != "" {
		toolMsg := llms.MessageContent{
			Role: llms.ChatMessageTypeSystem,
			Parts: []llms.ContentPart{
				llms.TextPart(fmt.Sprintf("I used the '%s' tool to help with your request. Here's the result:\n\n%s", toolName, toolResult)),
			},
		}
		a.messages = append(a.messages, toolMsg)
	}

	// Call LLM with full history
	response, err := a.llm.GenerateContent(ctx, a.messages)
	if err != nil {
		return "", fmt.Errorf("LLM call failed: %w", err)
	}

	// Extract response text
	var responseText string
	if response != nil && len(response.Choices) > 0 {
		responseText = response.Choices[0].Content
	}

	// Add assistant response to history
	assistantMsg := llms.MessageContent{
		Role:  llms.ChatMessageTypeAI,
		Parts: []llms.ContentPart{llms.TextPart(responseText)},
	}
	a.messages = append(a.messages, assistantMsg)

	return responseText, nil
}

// getClientID generates a unique client ID based on IP and User-Agent
func getClientID(r *http.Request) string {
	// Get client IP
	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.Header.Get("X-Real-IP")
	}
	if clientIP == "" {
		clientIP = strings.Split(r.RemoteAddr, ":")[0]
	}

	// Get User-Agent
	userAgent := r.Header.Get("User-Agent")
	if userAgent == "" {
		userAgent = "unknown"
	}

	// Create unique hash from IP + User-Agent
	h := md5.New()
	h.Write([]byte(clientIP + userAgent + "chat-salt"))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// ChatServer manages HTTP endpoints and chat agents
type ChatServer struct {
	sessionManager  *SessionManager
	agents          map[string]ChatAgent
	llm             llms.Model
	agentMu         sync.RWMutex
	port            string
	config          Config
	sessionManagers map[string]*SessionManager // clientID -> SessionManager
	smMu            sync.RWMutex
}

// NewChatServer creates a new chat server
func NewChatServer(sessionDir string, maxHistory int, port string) (*ChatServer, error) {
	// Load environment variables from .env file
	loadEnv()

	// Get configuration
	config := getConfig()

	// Check API key
	if config.OpenAIAPIKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create OpenAI LLM (works with OpenAI-compatible APIs like Baidu)
	var llm llms.Model
	var err error

	if config.OpenAIBaseURL != "" {
		llm, err = openai.New(
			openai.WithModel(config.OpenAIModel),
			openai.WithToken(config.OpenAIAPIKey),
			openai.WithBaseURL(config.OpenAIBaseURL),
		)
	} else {
		llm, err = openai.New(
			openai.WithModel(config.OpenAIModel),
			openai.WithToken(config.OpenAIAPIKey),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create LLM: %w", err)
	}

	return &ChatServer{
		sessionManager:  NewSessionManager(sessionDir, maxHistory),
		agents:          make(map[string]ChatAgent),
		llm:             llm,
		port:            port,
		config:          config,
		sessionManagers: make(map[string]*SessionManager),
	}, nil
}

// getSessionManager gets or creates a SessionManager for a specific client
func (cs *ChatServer) getSessionManager(clientID string) *SessionManager {
	cs.smMu.Lock()
	defer cs.smMu.Unlock()

	sm, exists := cs.sessionManagers[clientID]
	if !exists {
		clientSessionDir := fmt.Sprintf("%s/clients/%s", cs.sessionManager.sessionDir, clientID)
		sm = NewSessionManager(clientSessionDir, cs.sessionManager.maxHistory)
		cs.sessionManagers[clientID] = sm
	}
	return sm
}

// getOrCreateAgent gets an existing agent or creates a new one for a session
func (cs *ChatServer) getOrCreateAgent(sessionID string) (ChatAgent, error) {
	cs.agentMu.RLock()
	agent, exists := cs.agents[sessionID]
	cs.agentMu.RUnlock()

	if exists {
		return agent, nil
	}

	// Create new agent
	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	// Double-check after acquiring write lock
	if agent, exists := cs.agents[sessionID]; exists {
		return agent, nil
	}

	// Create simple chat agent
	agent = NewSimpleChatAgent(cs.llm, cs.config)
	cs.agents[sessionID] = agent

	// Initialize tools asynchronously to avoid blocking
	agent.(*SimpleChatAgent).InitializeToolsAsync()

	return agent, nil
}

// handleIndex serves the main HTML page
func (cs *ChatServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFile(w, r, "static/index.html")
}

// handleNewSession creates a new chat session
func (cs *ChatServer) handleNewSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)
	session := sm.CreateSession()

	// Set client ID cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "client_id",
		Value:    clientID,
		Path:     "/",
		MaxAge:   86400 * 30, // 30 days
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": session.ID,
		"client_id":  clientID,
	})
}

// handleListSessions returns all active sessions for the client
func (cs *ChatServer) handleListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)
	sessions := sm.ListSessions()

	type SessionInfo struct {
		ID           string    `json:"id"`
		Title        string    `json:"title"`
		MessageCount int       `json:"message_count"`
		CreatedAt    time.Time `json:"created_at"`
		UpdatedAt    time.Time `json:"updated_at"`
	}

	sessionInfos := make([]SessionInfo, 0, len(sessions))
	for _, session := range sessions {
		// Get the first user message as title
		title := "新会话"
		for _, msg := range session.Messages {
			if msg.Role == "user" {
				// Convert string to rune slice to properly handle UTF-8 characters
				runes := []rune(msg.Content)
				if len(runes) > 20 {
					title = string(runes[:20]) + "..."
				} else {
					title = msg.Content
				}
				break
			}
		}

		sessionInfos = append(sessionInfos, SessionInfo{
			ID:           session.ID,
			Title:        title,
			MessageCount: len(session.Messages),
			CreatedAt:    session.CreatedAt,
			UpdatedAt:    session.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(sessionInfos)
}

// handleDeleteSession deletes a session
func (cs *ChatServer) handleDeleteSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	// Close and delete agent
	cs.agentMu.Lock()
	if agent, exists := cs.agents[sessionID]; exists {
		// Close agent if it implements Close method
		log.Printf("Closing agent for deleted session %s", sessionID)
		if simpleAgent, ok := agent.(*SimpleChatAgent); ok {
			// Use a goroutine with timeout to prevent blocking
			done := make(chan error, 1)
			go func() {
				done <- simpleAgent.Close()
			}()

			// Wait for close with timeout
			select {
			case err := <-done:
				if err != nil {
					log.Printf("Error closing agent for session %s: %v", sessionID, err)
				}
			case <-time.After(10 * time.Second):
				log.Printf("Warning: Agent close for session %s timed out", sessionID)
			}
		}
		delete(cs.agents, sessionID)
		log.Printf("Agent for session %s deleted", sessionID)
	}
	cs.agentMu.Unlock()

	// Delete session
	err := sm.DeleteSession(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetHistory retrieves chat history for a session
func (cs *ChatServer) handleGetHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	sessionID := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
	sessionID = strings.TrimSuffix(sessionID, "/history")
	if sessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	messages, err := sm.GetMessages(sessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// handleChat handles chat message requests
func (cs *ChatServer) handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID    string `json:"session_id"`
		Message      string `json:"message"`
		UserSettings struct {
			EnableSkills bool `json:"enable_skills"`
			EnableMCP    bool `json:"enable_mcp"`
		} `json:"user_settings"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Failed to decode request: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.SessionID == "" || req.Message == "" {
		http.Error(w, "session_id and message are required", http.StatusBadRequest)
		return
	}

	clientID := getClientID(r)
	sm := cs.getSessionManager(clientID)

	log.Printf("Chat request for session %s: %s", req.SessionID, req.Message)

	// Verify session exists
	_, err := sm.GetSession(req.SessionID)
	if err != nil {
		log.Printf("Session not found: %s", req.SessionID)
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Get or create agent for this session
	agent, err := cs.getOrCreateAgent(req.SessionID)
	if err != nil {
		log.Printf("Failed to create agent: %v", err)
		http.Error(w, fmt.Sprintf("Failed to create agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Add user message to history
	sm.AddMessage(req.SessionID, "user", req.Message)

	// Get response from agent
	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	// Use user settings directly
	enableSkills := req.UserSettings.EnableSkills
	enableMCP := req.UserSettings.EnableMCP

	log.Printf("Tool settings for session %s - Skills: %v, MCP: %v",
		req.SessionID, enableSkills, enableMCP)

	response, err := agent.Chat(ctx, req.Message, enableSkills, enableMCP)
	if err != nil {
		log.Printf("Chat error for session %s: %v", req.SessionID, err)
		http.Error(w, fmt.Sprintf("Chat failed: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("Chat response for session %s: %s", req.SessionID, response)

	// Add assistant response to history
	sm.AddMessage(req.SessionID, "assistant", response)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

// handleGetClientID returns the client ID for the current user
func (cs *ChatServer) handleGetClientID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	clientID := getClientID(r)

	// Set client ID cookie if not already set
	_, err := r.Cookie("client_id")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:     "client_id",
			Value:    clientID,
			Path:     "/",
			MaxAge:   86400 * 30, // 30 days
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"client_id": clientID,
	})
}

// handleMCPTools returns the list of available MCP tools
func (cs *ChatServer) handleMCPTools(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Get or create agent for this session
	agent, err := cs.getOrCreateAgent(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Cast to SimpleChatAgent to access MCP methods
	simpleAgent, ok := agent.(*SimpleChatAgent)
	if !ok {
		http.Error(w, "Agent does not support MCP", http.StatusInternalServerError)
		return
	}

	tools := simpleAgent.GetAvailableTools()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tools":   tools,
		"enabled": simpleAgent.toolsEnabled,
	})
}

// handleToolsHierarchical returns tools in a hierarchical structure
func (cs *ChatServer) handleToolsHierarchical(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}

	// Get or create agent for this session
	agent, err := cs.getOrCreateAgent(sessionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get agent: %v", err), http.StatusInternalServerError)
		return
	}

	// Cast to SimpleChatAgent
	simpleAgent, ok := agent.(*SimpleChatAgent)
	if !ok {
		http.Error(w, "Agent does not support tools", http.StatusInternalServerError)
		return
	}

	// Prepare hierarchical data
	var result struct {
		Skills       []map[string]interface{} `json:"skills"`
		MCPTools     []map[string]interface{} `json:"mcp_tools"`
		Enabled      bool                     `json:"enabled"`
		ToolsLoading bool                     `json:"tools_loading"`
		ToolsLoaded  bool                     `json:"tools_loaded"`
	}

	// Lock for reading skills and MCP tools
	simpleAgent.mu.RLock()
	result.Enabled = simpleAgent.toolsEnabled
	result.ToolsLoading = simpleAgent.toolsLoading
	result.ToolsLoaded = simpleAgent.toolsLoaded
	skills := make([]SkillInfo, len(simpleAgent.skills))
	copy(skills, simpleAgent.skills)
	mcpTools := make([]tools.Tool, len(simpleAgent.mcpTools))
	copy(mcpTools, simpleAgent.mcpTools)
	simpleAgent.mu.RUnlock()

	// Add skills with their tools
	for _, skill := range skills {
		skillData := map[string]interface{}{
			"name":        skill.Name,
			"description": skill.Description,
			"tools":       []map[string]interface{}{},
		}

		// Get tools for this skill if already loaded
		if skill.Loaded && len(skill.Tools) > 0 {
			for _, tool := range skill.Tools {
				skillData["tools"] = append(skillData["tools"].([]map[string]interface{}), map[string]interface{}{
					"name":        tool.Name(),
					"description": tool.Description(),
				})
			}
		} else {
			// Load tools on demand
			if tools, err := simpleAgent.loadSkillTools(skill.Name); err == nil {
				for _, tool := range tools {
					skillData["tools"] = append(skillData["tools"].([]map[string]interface{}), map[string]interface{}{
						"name":        tool.Name(),
						"description": tool.Description(),
					})
				}
			}
		}

		result.Skills = append(result.Skills, skillData)
	}

	// Add MCP tools (group them by category if possible, or list them individually)
	mcpGroups := make(map[string][]map[string]interface{})
	for _, tool := range mcpTools {
		toolName := tool.Name()
		desc := tool.Description()

		// Try to extract category from tool name (e.g., "puppeteer__puppeteer_navigate" -> "Puppeteer")
		parts := strings.Split(toolName, "__")
		var category string
		if len(parts) >= 2 {
			// Convert first letter to uppercase
			category = strings.ToUpper(parts[0][:1]) + strings.ToLower(parts[0][1:])
		} else {
			category = "Other"
		}

		mcpGroups[category] = append(mcpGroups[category], map[string]interface{}{
			"name":        toolName,
			"description": desc,
		})
	}

	// Convert groups to array
	for category, tools := range mcpGroups {
		result.MCPTools = append(result.MCPTools, map[string]interface{}{
			"category":     category,
			"description":  fmt.Sprintf("%s tools", category),
			"tools":        tools,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// handleConfig returns the chat configuration
func (cs *ChatServer) handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"chatTitle": cs.config.ChatTitle,
	})
}

// Close gracefully shuts down the server and cleans up all resources
func (cs *ChatServer) Close() error {
	log.Printf("Shutting down chat server...")

	cs.agentMu.Lock()
	defer cs.agentMu.Unlock()

	// Close all agents with error collection
	var closeErrors []error
	for sessionID, agent := range cs.agents {
		log.Printf("Closing agent for session %s", sessionID)
		if simpleAgent, ok := agent.(*SimpleChatAgent); ok {
			if err := simpleAgent.Close(); err != nil {
				log.Printf("Error closing agent for session %s: %v", sessionID, err)
				closeErrors = append(closeErrors, fmt.Errorf("session %s: %w", sessionID, err))
			}
		}
	}

	// Clear agents map
	cs.agents = make(map[string]ChatAgent)

	if len(closeErrors) > 0 {
		log.Printf("Chat server shutdown completed with %d errors", len(closeErrors))
		// Return first error but log all
		return closeErrors[0]
	}

	log.Printf("Chat server shutdown complete")
	return nil
}

// Start starts the HTTP server
func (cs *ChatServer) Start() error {
	http.HandleFunc("/", cs.handleIndex)
	http.HandleFunc("/api/client-id", cs.handleGetClientID)
	http.HandleFunc("/api/sessions/new", cs.handleNewSession)
	http.HandleFunc("/api/sessions", cs.handleListSessions)
	http.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasSuffix(path, "/history") {
			cs.handleGetHistory(w, r)
		} else if r.Method == http.MethodDelete {
			cs.handleDeleteSession(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	http.HandleFunc("/api/chat", cs.handleChat)
	http.HandleFunc("/api/mcp/tools", cs.handleMCPTools)
	http.HandleFunc("/api/tools/hierarchical", cs.handleToolsHierarchical)
	http.HandleFunc("/api/config", cs.handleConfig)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	addr := ":" + cs.port
	log.Printf("Chat server starting on http://localhost%s", addr)
	return http.ListenAndServe(addr, nil)
}

// getSkillsOverview returns a formatted string of available skills (name and description only)
func (a *SimpleChatAgent) getSkillsOverview() string {
	if len(a.skills) == 0 {
		return ""
	}

	var info strings.Builder
	info.WriteString("Available Skills:\n\n")

	for _, skill := range a.skills {
		info.WriteString(fmt.Sprintf("- %s: %s\n", skill.Name, skill.Description))
	}

	return info.String()
}

// loadSkillTools loads and caches tools for a specific skill
func (a *SimpleChatAgent) loadSkillTools(skillName string) ([]tools.Tool, error) {
	// Find the skill
	for i := range a.skills {
		if strings.EqualFold(a.skills[i].Name, skillName) {
			if !a.skills[i].Loaded {
				// Convert skill to tools
				skillTools, err := adaptergoskills.SkillsToTools(*a.skills[i].Package)
				if err != nil {
					return nil, fmt.Errorf("failed to convert skill '%s' to tools: %w", skillName, err)
				}
				a.skills[i].Tools = skillTools
				a.skills[i].Loaded = true
				log.Printf("Loaded %d tools from skill '%s'", len(skillTools), skillName)
			}
			return a.skills[i].Tools, nil
		}
	}
	return nil, fmt.Errorf("skill '%s' not found", skillName)
}

// selectSkillForTask uses LLM to determine which skill (if any) should be used for the task
func (a *SimpleChatAgent) selectSkillForTask(ctx context.Context, message string) (string, error) {
	if len(a.skills) == 0 {
		return "", nil // No skills available
	}

	skillsOverview := a.getSkillsOverview()

	skillPrompt := fmt.Sprintf(`Based on the user's message, determine if any of the available skills should be used to help with this task.

%s

User message: %s

Respond with a JSON object:
- If no skill is needed: {"use_skill": false, "reason": "reason why no skill is needed"}
- If a skill is needed: {"use_skill": true, "skill_name": "exact skill name", "reason": "why this skill is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use ` + "```json" + ` wrapper
- Choose the skill that best matches the user's needs`, skillsOverview, message)

	// Create LLM call for skill selection
	skillMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate skills for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(skillPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, skillMsg)
	if err != nil {
		return "", fmt.Errorf("LLM call failed for skill selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Skill selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if strings.HasPrefix(cleanDecision, "```json") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```json")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if strings.HasPrefix(cleanDecision, "```") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var skillDecision struct {
		UseSkill  bool   `json:"use_skill"`
		SkillName string `json:"skill_name"`
		Reason    string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &skillDecision); err != nil {
		return "", fmt.Errorf("failed to parse skill decision: %w", err)
	}

	if skillDecision.UseSkill {
		log.Printf("Selected skill '%s' because: %s", skillDecision.SkillName, skillDecision.Reason)
		return skillDecision.SkillName, nil
	}

	log.Printf("No skill selected: %s", skillDecision.Reason)
	return "", nil
}

// selectToolForTask uses LLM to determine which tool should be used
func (a *SimpleChatAgent) selectToolForTask(ctx context.Context, message string, availableTools []tools.Tool) (*tools.Tool, map[string]interface{}, error) {
	if len(availableTools) == 0 {
		return nil, nil, nil // No tools available
	}

	// Build tools info
	var toolsInfo strings.Builder
	for _, tool := range availableTools {
		toolsInfo.WriteString(fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description()))
	}

	toolPrompt := fmt.Sprintf(`Based on the user's message, determine which tool should be used.

Available tools:
%s

User message: %s

Respond with a JSON object:
- If no tool is needed: {"use_tool": false, "reason": "reason why no tool is needed"}
- If a tool is needed: {"use_tool": true, "tool_name": "exact tool name", "args": {parameter: "value"}, "reason": "why this tool is appropriate"}

IMPORTANT:
- Return ONLY valid JSON
- Do NOT use markdown code fences
- Do NOT use ` + "```json" + ` wrapper
- Select the tool that can best accomplish the user's request`, toolsInfo, message)

	// Create LLM call for tool selection
	toolMsg := []llms.MessageContent{
		{Role: llms.ChatMessageTypeSystem, Parts: []llms.ContentPart{llms.TextPart("You are a helpful assistant that selects appropriate tools for tasks. Respond only with valid JSON.")}},
		{Role: llms.ChatMessageTypeHuman, Parts: []llms.ContentPart{llms.TextPart(toolPrompt)}},
	}

	response, err := a.llm.GenerateContent(ctx, toolMsg)
	if err != nil {
		return nil, nil, fmt.Errorf("LLM call failed for tool selection: %w", err)
	}

	if len(response.Choices) == 0 {
		return nil, nil, fmt.Errorf("no response from LLM")
	}

	decision := response.Choices[0].Content
	log.Printf("Tool selection decision: %s", decision)

	// Clean up the decision
	cleanDecision := strings.TrimSpace(decision)
	if strings.HasPrefix(cleanDecision, "```json") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```json")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	} else if strings.HasPrefix(cleanDecision, "```") {
		cleanDecision = strings.TrimPrefix(cleanDecision, "```")
		cleanDecision = strings.TrimSuffix(cleanDecision, "```")
		cleanDecision = strings.TrimSpace(cleanDecision)
	}

	// Parse the decision
	var toolDecision struct {
		UseTool  bool                   `json:"use_tool"`
		ToolName string                 `json:"tool_name"`
		Args     map[string]interface{} `json:"args"`
		Reason   string                 `json:"reason"`
	}

	if err := json.Unmarshal([]byte(cleanDecision), &toolDecision); err != nil {
		return nil, nil, fmt.Errorf("failed to parse tool decision: %w", err)
	}

	if toolDecision.UseTool {
		// Find the selected tool
		for _, tool := range availableTools {
			if strings.EqualFold(tool.Name(), toolDecision.ToolName) {
				log.Printf("Selected tool '%s' because: %s", toolDecision.ToolName, toolDecision.Reason)
				return &tool, toolDecision.Args, nil
			}
		}
		return nil, nil, fmt.Errorf("tool '%s' not found in available tools", toolDecision.ToolName)
	}

	log.Printf("No tool selected: %s", toolDecision.Reason)
	return nil, nil, nil
}

func main() {
	// Load configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./sessions"
	}

	maxHistory := 50
	if maxHistoryStr := os.Getenv("MAX_HISTORY_SIZE"); maxHistoryStr != "" {
		fmt.Sscanf(maxHistoryStr, "%d", &maxHistory)
	}

	// Create and start server
	server, err := NewChatServer(sessionDir, maxHistory, port)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Setup graceful shutdown
	serverErr := make(chan error, 1)
	go func() {
		if err := server.Start(); err != nil {
			serverErr <- err
		}
	}()

	// Wait for interrupt signal or server error
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Block until signal received or server error
	select {
	case sig := <-sigChan:
		log.Printf("Received shutdown signal: %v", sig)
	case err := <-serverErr:
		log.Printf("Server error: %v", err)
	}

	// Graceful shutdown with timeout
	log.Println("Starting graceful shutdown...")
	shutdownDone := make(chan error, 1)
	go func() {
		shutdownDone <- server.Close()
	}()

	// Wait for shutdown to complete with timeout
	select {
	case err := <-shutdownDone:
		if err != nil {
			log.Printf("Error during shutdown: %v", err)
			os.Exit(1)
		}
		log.Println("Shutdown complete")
		os.Exit(0)
	case <-time.After(15 * time.Second):
		log.Println("Shutdown timed out after 15 seconds, forcing exit")
		os.Exit(1)
	}
}
