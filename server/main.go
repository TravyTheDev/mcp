package main

import (
	"encoding/json"
	"fmt"
	"log"
	"mcp-server/ai"

	"github.com/joho/godotenv"
)

type MCPMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
}

type ModelRequest struct {
	Prompt string `json:"prompt"`
}

type ModelResponse struct {
	Text string `json:"text"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// scanner := bufio.NewScanner(os.Stdin)
	// fmt.Fprintln(os.Stderr, "MCP Server started. Waiting for input...")
	// for scanner.Scan() {
	// 	// Read input message
	// 	input := scanner.Text()
	// 	// Parse the MCP message
	// 	var message MCPMessage
	// 	if err := json.Unmarshal([]byte(input), &message); err != nil {
	// 		sendError(err)
	// 		continue
	// 	}
	// 	// Process the message based on its type
	// 	switch message.Type {
	// 	case "request":
	// 		handleRequest(message.Content)
	// 	case "ping":
	// 		handlePing()
	// 	default:
	// 		sendError(fmt.Errorf("unknown message type: %s", message.Type))
	// 	}
	// }
	// if err := scanner.Err(); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
	// }
	ai.Run()
}

func handleRequest(content json.RawMessage) {
	var req ModelRequest
	if err := json.Unmarshal(content, &req); err != nil {
		sendError(err)
		return
	}
	// Call your model to generate text (simplified here)
	generatedText := generateText(req.Prompt)
	// Create and send response
	resp := ModelResponse{
		Text: generatedText,
	}
	respContent, err := json.Marshal(resp)
	if err != nil {
		sendError(err)
		return
	}
	respMessage := MCPMessage{
		Type:    "response",
		Content: respContent,
	}
	respData, err := json.Marshal(respMessage)
	if err != nil {
		sendError(err)
		return
	}
	fmt.Println(string(respData))
}

func handlePing() {
	// Respond to ping with a pong
	pongMessage := MCPMessage{
		Type:    "pong",
		Content: json.RawMessage(`{}`),
	}
	respData, err := json.Marshal(pongMessage)
	if err != nil {
		sendError(err)
		return
	}
	fmt.Println(string(respData))
}

func sendError(err error) {
	errorMsg := struct {
		Error string `json:"error"`
	}{
		Error: err.Error(),
	}
	content, _ := json.Marshal(errorMsg)
	errMessage := MCPMessage{
		Type:    "error",
		Content: content,
	}
	respData, _ := json.Marshal(errMessage)
	fmt.Println(string(respData))
}

func generateText(prompt string) string {
	// This is where you would integrate with your LLM
	// For this example, we'll just return a simple response
	return "This is a response to: " + prompt
}
