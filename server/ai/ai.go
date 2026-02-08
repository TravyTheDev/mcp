package ai

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

func Run() {
	ctx := context.Background()
	apiKey := os.Getenv("API_KEY")
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		log.Fatal(err)
	}
	result, _ := client.Models.GenerateContent(
		ctx,
		"gemini-3-flash-preview",
		genai.Text("Explain how AI works in a few words"),
		nil,
	)

	fmt.Println(result.Text())
}
