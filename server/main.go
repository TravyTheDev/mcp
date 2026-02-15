package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"mcp-server/db"
	"mcp-server/services/humans"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

// MCP JSON-RPC Types
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	log.SetOutput(os.Stderr)

	database, err := db.NewSqlStorage()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()
	humanStore := humans.NewHumanStore(database)

	reader := bufio.NewReader(os.Stdin)
	for {

		input, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading input: %v", err)
			}
			break
		}

		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			log.Printf("Failed to unmarshal request: %v | Raw: %q", err, input)
			continue
		}

		switch req.Method {
		case "initialize":
			sendResponse(req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]string{"name": "human-finder", "version": "1.0.0"},
			})

		case "tools/list":
			sendResponse(req.ID, map[string]any{
				"tools": []map[string]any{
					{
						"name":        "get_humans",
						"description": "Returns all humans from the database",
						"inputSchema": map[string]any{
							"type":       "object",
							"properties": map[string]any{},
						},
					},
				},
			})

		case "tools/call":
			var params CallToolParams
			json.Unmarshal(req.Params, &params)

			if params.Name == "get_humans" {
				data, err := humanStore.GetHumans()
				if err != nil {
					sendError(req.ID, -32603, err.Error())
				} else {
					sendResponse(req.ID, map[string]any{
						"content": []map[string]any{
							{
								"type": "text",
								"text": formatHumans(data),
							},
						},
					})
				}
			} else {
				sendError(req.ID, -32601, "Tool not found")
			}

		default:
			log.Printf("Received unknown method: %s", req.Method)
		}
	}
}

func sendResponse(id any, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

func sendError(id any, code int, message string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	json.NewEncoder(os.Stdout).Encode(resp)
}

func formatHumans(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
}
