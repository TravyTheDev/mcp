package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/joho/godotenv"
	"google.golang.org/genai"
	_ "modernc.org/sqlite"
)

type MCPToolListResult struct {
	Tools []struct {
		Name        string                 `json:"name"`
		Description string                 `json:"description"`
		InputSchema map[string]interface{} `json:"inputSchema"`
	} `json:"tools"`
}

type JSONRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type JSONRPCResponse struct {
	Result json.RawMessage `json:"result"`
	Error  *any            `json:"error"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	ctx := context.Background()
	prompt := "I am a cat, which human is best for me?"

	cmd := exec.Command("../server/mcp-server")
	cmd.Dir = "../server"

	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Process.Kill()

	sendRequest(stdin, "initialize", map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]string{"name": "my-fun-client", "version": "1.0.0"},
	}, 1)
	readResponse(stdout)

	sendRequest(stdin, "tools/list", map[string]any{}, 2)
	toolData := readResponse(stdout)

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  os.Getenv("API_KEY"),
		Backend: genai.BackendGeminiAPI,
	})

	if err != nil {
		log.Fatalf("Failed to create Gemini client: %v", err)
	}

	if toolData == nil {
		log.Fatal("Server failed to provide tools. Check server logs (Stderr) above.")
	}

	geminiTools := convertMcpToGemini(toolData)

	runGeminiLoop(ctx, client, stdin, stdout, prompt, geminiTools)
}

func runGeminiLoop(ctx context.Context, client *genai.Client, serverIn io.WriteCloser, serverOut io.Reader, prompt string, tools []*genai.Tool) {
	model := "gemini-3-flash-preview"
	history := []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: prompt}}}}

	for {
		resp, err := client.Models.GenerateContent(ctx, model, history, &genai.GenerateContentConfig{Tools: tools})
		if err != nil {
			log.Fatal(err)
		}

		candidate := resp.Candidates[0]
		history = append(history, candidate.Content)

		var toolCall *genai.FunctionCall
		for _, part := range candidate.Content.Parts {
			if part.FunctionCall != nil {
				toolCall = part.FunctionCall
			}
		}

		if toolCall == nil {
			fmt.Println("\nGemini:", resp.Text())
			break
		}

		fmt.Printf("--- Gemini calling tool: %s ---\n", toolCall.Name)

		sendRequest(serverIn, "tools/call", map[string]any{
			"name":      toolCall.Name,
			"arguments": toolCall.Args,
		}, 3)

		toolResultRaw := readResponse(serverOut)

		var mcpResult struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		json.Unmarshal(toolResultRaw, &mcpResult)

		history = append(history, &genai.Content{
			Role: "tool",
			Parts: []*genai.Part{{
				FunctionResponse: &genai.FunctionResponse{
					Name:     toolCall.Name,
					Response: map[string]any{"result": mcpResult.Content[0].Text},
				},
			}},
		})
	}
}

func sendRequest(w io.Writer, method string, params any, id int) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}
	json.NewEncoder(w).Encode(req)
}

func readResponse(r io.Reader) json.RawMessage {
	scanner := bufio.NewScanner(r)
	if scanner.Scan() {
		var resp JSONRPCResponse
		json.Unmarshal(scanner.Bytes(), &resp)
		return resp.Result
	}
	return nil
}

func convertMcpToGemini(toolData json.RawMessage) []*genai.Tool {
	var result MCPToolListResult
	err := json.Unmarshal(toolData, &result)
	if err != nil {
		log.Printf("Error parsing tools from server: %v", err)
		return nil
	}
	var declarations []*genai.FunctionDeclaration

	for _, mcpTool := range result.Tools {
		decl := &genai.FunctionDeclaration{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
		}

		// Map the JSON Schema to Gemini Schema
		if mcpTool.InputSchema != nil {
			decl.Parameters = mapJsonSchemaToGemini(mcpTool.InputSchema)
		}

		declarations = append(declarations, decl)
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}
}

func mapJsonSchemaToGemini(schema map[string]interface{}) *genai.Schema {
	// Simple version: MCP Tools usually define an "object" at the top level
	gSchema := &genai.Schema{
		Type: genai.TypeObject,
	}

	properties, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return gSchema
	}

	gSchema.Properties = make(map[string]*genai.Schema)
	for name, details := range properties {
		propMap := details.(map[string]interface{})

		// Map JSON types (string, number, etc) to Gemini Types
		var propType genai.Type
		switch propMap["type"] {
		case "string":
			propType = genai.TypeString
		case "number", "integer":
			propType = genai.TypeNumber
		case "boolean":
			propType = genai.TypeBoolean
		default:
			propType = genai.TypeString
		}

		gSchema.Properties[name] = &genai.Schema{
			Type:        propType,
			Description: fmt.Sprintf("%v", propMap["description"]),
		}
	}

	return gSchema
}
