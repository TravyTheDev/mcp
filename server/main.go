package main

import (
	"encoding/json"
	"io"
	"log"
	"mcp-server/db"
	"mcp-server/services/humans"
	"net/http"

	"github.com/joho/godotenv"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
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

	database, err := db.NewSqlStorage()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()
	humanStore := humans.NewHumanStore(database)

	http.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		var req JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		switch req.Method {
		case "initialize":
			sendResponse(w, req.ID, map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]string{"name": "human-finder", "version": "1.0.0"},
			})

		case "tools/list":
			sendResponse(w, req.ID, map[string]any{
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
					sendError(w, req.ID, -32603, err.Error())
				} else {
					sendResponse(w, req.ID, map[string]any{
						"content": []map[string]any{
							{
								"type": "text",
								"text": formatHumans(data),
							},
						},
					})
				}
			} else {
				sendError(w, req.ID, -32601, "Tool not found")
			}

		default:
			log.Printf("Received unknown method: %s", req.Method)
		}
	})
	log.Println("Server running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendResponse(w io.Writer, id any, result any) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
	json.NewEncoder(w).Encode(resp)
}

func sendError(w io.Writer, id any, code int, message string) {
	resp := map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func formatHumans(data any) string {
	b, _ := json.Marshal(data)
	return string(b)
}
