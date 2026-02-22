package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mcp-server/db"
	"mcp-server/services/humans"
	"net/http"
	"os"

	"github.com/rs/cors"
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
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Fatal("Error loading .env file")
	// }

	frontUrl := os.Getenv("FRONT_URL")
	frontUrlWWW := os.Getenv("FRONT_URL_WWW")
	PORT := os.Getenv("PORT")

	router := http.NewServeMux()
	database, err := db.NewSqlStorage()
	if err != nil {
		log.Fatalf("Failed to connect to DB: %v", err)
	}
	defer database.Close()

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

	humanStore := humans.NewHumanStore(database)

	router.HandleFunc("/mcp_api/load_humans", func(w http.ResponseWriter, r *http.Request) {
		res, err := humanStore.GetHumans()
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			fmt.Println(err)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Header().Add("Content-Type", "application/json")

		if err := json.NewEncoder(w).Encode(res); err != nil {
			http.Error(w, "error getting user", http.StatusInternalServerError)
			fmt.Println(err)
			return
		}
	})

	router.HandleFunc("/mcp_api/client_request", func(w http.ResponseWriter, r *http.Request) {
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
	log.Fatal(server.ListenAndServe())
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
