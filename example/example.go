package main

import (
	"context"
	"fmt"
	"log"

	llmstructed "github.com/glidea/llm-structed"
)

type Summary struct {
	Title    string   `json:"title" desc:"The title of the summary"`
	Content  string   `json:"content" desc:"A concise summary of the article content"`
	Keywords []string `json:"keywords" desc:"Key topics mentioned in the article"`
	Score    int      `json:"score" desc:"The quality score of the article (1-10)"`
	Category string   `json:"category" desc:"The category of the article" enum:"Technology,Science,Business,Health,Education,Other"`
}

func main() {
	client, err := llmstructed.New(llmstructed.Config{
		BaseURL:                   "https://openrouter.ai/api/v1",
		APIKey:                    "sk-...",
		Model:                     "google/gemini-flash-1.5",
		Temperature:               0.3,
		StructuredOutputSupported: true, // try switch it when structured result is unexpected
		Retry:                     1,
		Debug:                     true,
	})
	if err != nil {
		log.Fatalf("Failed to create LLM client: %v", err)
	}

	article := `Artificial Intelligence (AI) is transforming the way we live and work. It refers to
	computer systems that can perform tasks that normally require human intelligence. These
	tasks include visual perception, speech recognition, decision-making, and language
	translation. Machine learning, a subset of AI, enables systems to learn and improve
	from experience without being explicitly programmed. Deep learning, particularly,
	has revolutionized AI by using neural networks to process complex patterns in data.`

	var summary Summary
	if err := client.Do(context.Background(), []string{
		"Please generate a summary of this article: " + article,
	}, &summary); err != nil {
		log.Fatalf("Failed to generate summary: %v", err)
	}

	fmt.Println("--------------------------------")
	fmt.Println("Title: ", summary.Title)
	fmt.Println("Summary: ", summary.Content)
	fmt.Println("Keywords: ", summary.Keywords)
	fmt.Println("Score: ", summary.Score)
	fmt.Println("Category: ", summary.Category)

	// Unstructured output (like other general clients)
	content, _ := client.DoUnstructured(context.Background(), []string{
		"Hello, who are you?",
	})
	fmt.Println("Content: ", content)
}
