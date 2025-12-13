package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/smallnest/langgraphgo/graph"
	"github.com/smallnest/langgraphgo/memory"
)

// ConversationState represents the state of our conversational agent
type ConversationState struct {
	UserInput      string
	Intent         string
	Context        []*memory.Message
	Response       string
	Memory         memory.Memory
	ConversationID string
	TurnCount      int
}

// IntentClassifier analyzes user input and determines intent
func classifyIntent(ctx context.Context, state any) (any, error) {
	s := state.(ConversationState)
	fmt.Println("\n[Node: Classify Intent]")

	mem := s.Memory
	userInput := s.UserInput

	// Get conversation context from memory
	context, _ := mem.GetContext(ctx, userInput)
	s.Context = context

	fmt.Printf("  User: %s\n", userInput)
	fmt.Printf("  Context retrieved: %d messages\n", len(context))

	// Simple intent classification based on keywords
	inputLower := strings.ToLower(userInput)
	var intent string

	if strings.Contains(inputLower, "price") || strings.Contains(inputLower, "cost") || strings.Contains(inputLower, "$") {
		intent = "pricing_query"
	} else if strings.Contains(inputLower, "feature") || strings.Contains(inputLower, "capability") || strings.Contains(inputLower, "can it") {
		intent = "feature_query"
	} else if strings.Contains(inputLower, "warranty") || strings.Contains(inputLower, "guarantee") {
		intent = "warranty_query"
	} else if strings.Contains(inputLower, "shipping") || strings.Contains(inputLower, "delivery") {
		intent = "shipping_query"
	} else if strings.Contains(inputLower, "hello") || strings.Contains(inputLower, "hi") {
		intent = "greeting"
	} else if strings.Contains(inputLower, "my name") || strings.Contains(inputLower, "i am") || strings.Contains(inputLower, "i'm") {
		intent = "introduction"
	} else if strings.Contains(inputLower, "remember") || strings.Contains(inputLower, "recall") {
		intent = "memory_check"
	} else {
		intent = "general_query"
	}

	s.Intent = intent
	fmt.Printf("  Detected intent: %s\n", intent)

	return s, nil
}

// RetrieveInformation retrieves relevant information based on intent
func retrieveInformation(ctx context.Context, state any) (any, error) {
	s := state.(ConversationState)
	fmt.Println("\n[Node: Retrieve Information]")

	intent := s.Intent
	context := s.Context
	mem := s.Memory

	fmt.Printf("  Intent: %s\n", intent)
	fmt.Printf("  Available context: %d messages\n", len(context))

	// Check if we have relevant information in context
	var foundInfo bool
	for _, msg := range context {
		msgLower := strings.ToLower(msg.Content)
		if intent == "pricing_query" && strings.Contains(msgLower, "$") {
			foundInfo = true
			break
		}
	}

	if foundInfo {
		fmt.Printf("  Found relevant info in context\n")
	} else {
		fmt.Printf("  No previous info, will use knowledge base\n")
	}

	// Get memory stats
	stats, _ := mem.GetStats(ctx)
	if stats != nil {
		fmt.Printf("  Memory stats: %d total messages, %d active\n", stats.TotalMessages, stats.ActiveMessages)
	}

	return s, nil
}

// GenerateResponse creates a response based on intent and context
func generateResponse(ctx context.Context, state any) (any, error) {
	s := state.(ConversationState)
	fmt.Println("\n[Node: Generate Response]")

	userInput := s.UserInput
	intent := s.Intent
	context := s.Context
	mem := s.Memory

	var response string

	// Generate response based on intent and context
	switch intent {
	case "greeting":
		response = "Hello! I'm your product assistant. How can I help you today?"

	case "introduction":
		// Extract name
		words := strings.Fields(userInput)
		var name string
		for i, word := range words {
			if (strings.ToLower(word) == "am" || strings.ToLower(word) == "i'm" || strings.Contains(strings.ToLower(word), "name")) && i+1 < len(words) {
				name = words[i+1]
				break
			}
		}
		if name != "" {
			response = fmt.Sprintf("Nice to meet you, %s! I'll remember your name.", name)
		} else {
			response = "Nice to meet you! How can I assist you today?"
		}

	case "pricing_query":
		// Check if price was mentioned before
		priceKnown := false
		for _, msg := range context {
			if strings.Contains(msg.Content, "$99") {
				priceKnown = true
				break
			}
		}
		if priceKnown {
			response = "As I mentioned before, our premium product is priced at $99 with free shipping!"
		} else {
			response = "Our premium product is priced at $99, which includes free shipping and a 2-year warranty!"
		}

	case "feature_query":
		response = "Our product has amazing features: waterproof design, 24-hour battery life, AI-powered assistance, and wireless charging!"

	case "warranty_query":
		response = "Yes! We offer a comprehensive 2-year warranty covering all manufacturing defects and free replacement."

	case "shipping_query":
		response = "We offer free standard shipping (3-5 business days) and express shipping ($15, 1-2 days) worldwide!"

	case "memory_check":
		// Check what we remember
		var userName string
		for _, msg := range context {
			if msg.Role == "user" && (strings.Contains(msg.Content, "I am") || strings.Contains(msg.Content, "My name")) {
				words := strings.Fields(msg.Content)
				for i, word := range words {
					if (strings.ToLower(word) == "am" || strings.Contains(strings.ToLower(word), "name")) && i+1 < len(words) {
						userName = words[i+1]
						break
					}
				}
			}
		}
		if userName != "" {
			response = fmt.Sprintf("Of course! I remember you, %s. I have our conversation history with %d messages.", userName, len(context))
		} else {
			response = fmt.Sprintf("I have our conversation history with %d messages. How can I help you?", len(context))
		}

	case "general_query":
		if len(context) > 2 {
			response = "Based on our conversation so far, I'm here to help! Could you please provide more details?"
		} else {
			response = "I'm here to help! Could you please tell me more about what you're looking for?"
		}

	default:
		response = "I understand. How can I assist you further?"
	}

	s.Response = response
	fmt.Printf("  Generated response: %s\n", response)

	// Add user message to memory
	userMsg := memory.NewMessage("user", userInput)
	mem.AddMessage(ctx, userMsg)

	// Add assistant response to memory
	assistantMsg := memory.NewMessage("assistant", response)
	mem.AddMessage(ctx, assistantMsg)

	// Update turn count
	s.TurnCount = s.TurnCount + 1
	fmt.Printf("  Conversation turn: %d\n", s.TurnCount)

	return s, nil
}

// CreateConversationGraph creates a graph with memory integration
func CreateConversationGraph() *graph.StateGraph {
	// Create workflow
	g := graph.NewStateGraph()

	// Add nodes
	g.AddNode("classify_intent", "classify_intent", classifyIntent)
	g.AddNode("retrieve_info", "retrieve_info", retrieveInformation)
	g.AddNode("generate_response", "generate_response", generateResponse)

	// Define edges
	g.AddEdge("classify_intent", "retrieve_info")
	g.AddEdge("retrieve_info", "generate_response")
	g.AddEdge("generate_response", graph.END)

	// Set entry point
	g.SetEntryPoint("classify_intent")

	return g
}

// Demo functions for different memory strategies

func demoSlidingWindow() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Demo 1: Sliding Window Memory with Graph")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Strategy: Sliding Window (keeps last 4 messages)")
	fmt.Println("Scenario: Recent context is most important")

	mem := memory.NewSlidingWindowMemory(4)
	workflow := CreateConversationGraph()
	runnable, _ := workflow.Compile()

	conversations := []string{
		"Hello!",
		"What's the price?",
		"Tell me about features",
		"What's the warranty?",
		"Remind me of the price?", // Tests if price is still in window
	}

	for _, msg := range conversations {
		initialState := ConversationState{
			UserInput:      msg,
			Memory:         mem,
			ConversationID: "demo1",
		}

		result, err := runnable.Invoke(context.Background(), initialState)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		resultState := result.(ConversationState)
		fmt.Printf("\n→ Response: %s\n", resultState.Response)
		time.Sleep(300 * time.Millisecond)
	}

	stats, _ := mem.GetStats(context.Background())
	fmt.Printf("\n[Final Stats] Total: %d messages, Active: %d messages\n", stats.TotalMessages, stats.ActiveMessages)
}

func demoHierarchical() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Demo 2: Hierarchical Memory with Graph")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Strategy: Hierarchical (keeps important + recent)")
	fmt.Println("Scenario: Some information is more important")

	mem := memory.NewHierarchicalMemory(&memory.HierarchicalConfig{
		RecentLimit:    2,
		ImportantLimit: 3,
	})
	workflow := CreateConversationGraph()
	runnable, _ := workflow.Compile()

	conversations := []struct {
		text      string
		important bool
	}{
		{"Hi, my name is Alice", true},
		{"What's the price?", false},
		{"Tell me about features", false},
		{"IMPORTANT: I need waterproof capability", true},
		{"What about warranty?", false},
		{"What about shipping?", false},
		{"Do you remember my name?", false}, // Tests long-term memory
		{"And my waterproof requirement?", false},
	}

	for _, conv := range conversations {
		initialState := ConversationState{
			UserInput:      conv.text,
			Memory:         mem,
			ConversationID: "demo2",
		}

		result, err := runnable.Invoke(context.Background(), initialState)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		resultState := result.(ConversationState)
		fmt.Printf("\n→ Response: %s\n", resultState.Response)
		time.Sleep(300 * time.Millisecond)
	}

	stats, _ := mem.GetStats(context.Background())
	fmt.Printf("\n[Final Stats] Total: %d messages, Active: %d messages\n", stats.TotalMessages, stats.ActiveMessages)
}

func demoRetrieval() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Demo 3: Retrieval Memory with Graph")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Strategy: Retrieval (finds relevant messages)")
	fmt.Println("Scenario: Large conversation, query-driven")

	mem := memory.NewRetrievalMemory(&memory.RetrievalConfig{
		TopK: 3,
	})
	workflow := CreateConversationGraph()
	runnable, _ := workflow.Compile()

	conversations := []string{
		"Hi there!",
		"What's the price?",
		"Tell me about features",
		"Waterproof capability?",
		"What about warranty?",
		"Shipping options?",
		"Available colors?",
		"Battery life?",
		"Let's talk about the price again", // Should retrieve price-related messages
	}

	for _, msg := range conversations {
		initialState := ConversationState{
			UserInput:      msg,
			Memory:         mem,
			ConversationID: "demo3",
		}

		result, err := runnable.Invoke(context.Background(), initialState)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		resultState := result.(ConversationState)
		fmt.Printf("\n→ Response: %s\n", resultState.Response)
		time.Sleep(300 * time.Millisecond)
	}

	stats, _ := mem.GetStats(context.Background())
	fmt.Printf("\n[Final Stats] Total: %d messages stored\n", stats.TotalMessages)
}

func demoGraphMemory() {
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Demo 4: Graph-Based Memory with Graph Workflow")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println("Strategy: Graph-Based (tracks topic relationships)")
	fmt.Println("Scenario: Related topics and connections")

	mem := memory.NewGraphBasedMemory(&memory.GraphConfig{
		TopK: 4,
	})
	workflow := CreateConversationGraph()
	runnable, _ := workflow.Compile()

	conversations := []string{
		"What's the price?",
		"Tell me about the warranty",
		"Does the price include warranty?",   // Connects price + warranty
		"What features justify this price?",  // Connects features + price
		"Is shipping included in the price?", // Connects shipping + price
	}

	for _, msg := range conversations {
		initialState := ConversationState{
			UserInput:      msg,
			Memory:         mem,
			ConversationID: "demo4",
		}

		result, err := runnable.Invoke(context.Background(), initialState)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			continue
		}

		resultState := result.(ConversationState)
		fmt.Printf("\n→ Response: %s\n", resultState.Response)

		// Show topic relationships
		relations := mem.GetRelationships()
		fmt.Printf("   [Topics tracked: %v]\n", getMapKeys(relations))

		time.Sleep(300 * time.Millisecond)
	}

	stats, _ := mem.GetStats(context.Background())
	fmt.Printf("\n[Final Stats] Total: %d messages\n", stats.TotalMessages)
}

func getMapKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func main() {
	fmt.Println("╔═══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║     Memory + LangGraph Integration Examples                      ║")
	fmt.Println("║     Demonstrating State-based Memory in Graph Workflows          ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════════════╝")

	// Run demos
	demoSlidingWindow()
	time.Sleep(1 * time.Second)

	demoHierarchical()
	time.Sleep(1 * time.Second)

	demoRetrieval()
	time.Sleep(1 * time.Second)

	demoGraphMemory()

	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("Summary: Memory Strategies in Graph Workflows")
	fmt.Println(strings.Repeat("=", 70))
	fmt.Println()
	fmt.Println("Key Insights:")
	fmt.Println("1. Memory Strategy lives in State - accessible to all nodes")
	fmt.Println("2. Each node can:")
	fmt.Println("   - Retrieve context from memory")
	fmt.Println("   - Add new messages to memory")
	fmt.Println("   - Get memory statistics")
	fmt.Println("3. Different strategies provide different context:")
	fmt.Println("   - Sliding Window: Recent messages only")
	fmt.Println("   - Hierarchical: Important + recent messages")
	fmt.Println("   - Retrieval: Query-relevant messages")
	fmt.Println("   - Graph: Topic-related messages")
	fmt.Println()
	fmt.Println("This pattern allows:")
	fmt.Println("✓ Stateful conversations across workflow nodes")
	fmt.Println("✓ Context-aware decision making in each node")
	fmt.Println("✓ Flexible memory management strategies")
	fmt.Println("✓ Scalable conversation handling")
	fmt.Println()
}
