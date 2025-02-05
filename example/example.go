package main

import (
	"context"
	"fmt"

	"github.com/glidea/llm-structed"
)

type Summary struct {
	Title    string   `json:"title" desc:"The title of the summary"`
	Content  string   `json:"content" desc:"A concise summary of the article content"`
	Keywords []string `json:"keywords" desc:"Key topics mentioned in the article"`
	Score    int      `json:"score" desc:"The quality score of the article (1-10)"`
	Category string   `json:"category" desc:"The category of the article" enum:"Technology,Science,Business,Health,Education,Other"`
}

func main() {
	// New client (In minimal configuration, you only need to set the APIKey)
	cli, _ := llmstructed.New(llmstructed.Config{
		BaseURL:                   "https://openrouter.ai/api/v1",
		APIKey:                    "sk-...",
		Model:                     "google/gemini-flash-1.5",
		Temperature:               0.3,
		StructuredOutputSupported: true,
		Retry:                     1,
		Debug:                     true,
		// See source code comments of llmstructed.Config for these config detail
	})
	ctx := context.Background()

	// Structured Outputed
	var summary Summary
	_ = cli.Do(ctx, []string{`Please generate a summary of this article: Artificial Intelligence (AI) is transforming the way we live and work. It refers to
	computer systems that can perform tasks that normally require human intelligence. These
	tasks include visual perception, speech recognition, decision-making, and language
	translation. Machine learning, a subset of AI, enables systems to learn and improve
	from experience without being explicitly programmed. Deep learning, particularly,
	has revolutionized AI by using neural networks to process complex patterns in data.`,
	}, &summary)
	fmt.Printf("Go Struct: %v\n\n", summary)

	// Simple method for single value
	str, _ := cli.String(ctx, []string{"Hello, who are you?"})
	fmt.Printf("String: %s\n\n", str)
	languages, _ := cli.StringSlice(ctx, []string{"List some popular programming languages."})
	fmt.Printf("String Slice: %v\n\n", languages)
	count, _ := cli.Int(ctx, []string{`How many words are in this sentence: "Hello world, this is a test."`})
	fmt.Printf("Integer: %d\n\n", count)
	yes, _ := cli.Bool(ctx, []string{"Are you happy?"})
	fmt.Printf("Boolean: %v\n\n", yes)
	trues, _ := cli.BoolSlice(ctx, []string{"Are these statements true? [\"The sky is blue\", \"Fish can fly\", \"Water is wet\"]"})
	fmt.Printf("Boolean Slice: %v\n\n", trues)
	pi, _ := cli.Float(ctx, []string{"What is the value of pi (to two decimal places)?"})
	fmt.Printf("Float: %.2f\n\n", pi)
}
