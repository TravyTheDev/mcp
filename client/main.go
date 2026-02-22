package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/rs/cors"
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

type ChatRequest struct {
	Prompt string `json:"prompt"`
}

func main() {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }
	router := http.NewServeMux()
	PORT := os.Getenv("PORT")
	frontUrl := os.Getenv("FRONT_URL")
	frontUrlWWW := os.Getenv("FRONT_URL_WWW")

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{frontUrl, frontUrlWWW},
		AllowCredentials: true,
		AllowedMethods:   []string{http.MethodOptions, http.MethodGet, http.MethodPost, http.MethodDelete, http.MethodPut, http.MethodPatch},
	})

	handler := c.Handler(router)
	server := http.Server{
		Addr:    PORT,
		Handler: handler,
	}
	ctx := context.Background()

	callMCPServer("initialize", map[string]any{
		"protocolVersion": "2024-11-05",
	})

	toolData := callMCPServer("tools/list", map[string]any{})

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

	router.HandleFunc("/mcp_client/chat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Use POST", 405)
			return
		}
		var chatRequest ChatRequest
		if err := json.NewDecoder(r.Body).Decode(&chatRequest); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("X-Accel-Buffering", "no")

		runGeminiLoop(w, ctx, client, chatRequest.Prompt, geminiTools)
	})
	log.Fatal(server.ListenAndServe())
}

func runGeminiLoop(w http.ResponseWriter, ctx context.Context, client *genai.Client, prompt string, tools []*genai.Tool) {
	flusher, _ := w.(http.Flusher)
	model := "gemini-3-flash-preview"
	history := []*genai.Content{{Role: "user", Parts: []*genai.Part{{Text: prompt}}}}

	for {
		fmt.Fprint(w, "Thinking... ")
		flusher.Flush()
		resp, err := client.Models.GenerateContent(ctx, model, history, &genai.GenerateContentConfig{Tools: tools})
		if err != nil {
			fmt.Fprintln(w, "Error calling Gemini:", err)
			return
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
			fmt.Fprintln(w, resp.Text())
			flusher.Flush()
			break
		}

		fmt.Fprintf(w, "[System: Consulting database via %s...]\n", toolCall.Name)
		flusher.Flush()

		toolResultRaw := callMCPServer("tools/call", map[string]any{
			"name":      toolCall.Name,
			"arguments": toolCall.Args,
		})

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

func callMCPServer(method string, params any) json.RawMessage {
	SERVER_PORT := os.Getenv("MCP_PORT")
	url := fmt.Sprintf("http://localhost%s/mcp_api/client_request", SERVER_PORT)

	reqPayload := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      1,
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		log.Printf("Error marshaling request: %v", err)
		return nil
	}
	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var r JSONRPCResponse
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil
	}

	if r.Error != nil {
		log.Printf("Server returned error: %v", r.Error)
		return nil
	}

	return r.Result
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

		if mcpTool.InputSchema != nil {
			decl.Parameters = mapJsonSchemaToGemini(mcpTool.InputSchema)
		}

		declarations = append(declarations, decl)
	}

	return []*genai.Tool{{FunctionDeclarations: declarations}}
}

func mapJsonSchemaToGemini(schema map[string]interface{}) *genai.Schema {
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
