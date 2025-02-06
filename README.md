[中文](README-zh.md)

# Background
In chat scenarios, models typically do not need to return structured data. However, in LLM application development, models are often viewed as API services that provide some atomic capabilities, and at this point, we want to receive a JSON directly. The common solutions are:

## 1. Emphasizing the output format directly in the Prompt
* Pros: Simple, no additional requirements for the model API
* Cons: Format is unstable, especially for less capable models

## 2. Using response_format: { type: "json_object" } + Prompt to specify the exact fields
* Pros: Always ensures a valid JSON is returned
* Cons: Fields are unstable, especially for less capable models

## 3. Using response_format: { type: "json_schema", json_schema: {"strict": true, "schema": ...} }
* Pros: Ensures a valid JSON is returned, and fields are stable
* Cons: Only supported by some models

## SDK
* The [SDK]( https://platform.openai.com/docs/guides/structured-outputs?example=structured-data#how-to-use) provided by OpenAI directly supports Class as a Response Format, but only for Python
* In [go-openai]( https://github.com/sashabaranov/go-openai), the usage is overly generic and cumbersome
* llm-structed is specifically optimized for structured scenarios, providing native support for solutions 3 and 2

# llm-structed

llm-structed is an LLM Client optimized for structured output scenarios:
* Automatically converts Go struct definitions into Response JSON Schema
* Automatically transforms LLM output into Go structs
* Friendly declarative configuration based on struct tags
* Lightweight
* Based on [Json Schema or Json Object](https://platform.openai.com/docs/guides/structured-outputs#supported-schemas)
* Only support OpenAI compatible LLM, most mainstream providers have corresponding compatible interfaces, such as [Gemini](https://ai.google.dev/gemini-api/docs/openai)

## Installation

```bash
go get github.com/glidea/llm-structed
```

## Quick Start
```go
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
```

## Differences from [go-openapi](https://github.com/sashabaranov/go-openai)

* More lightweight
* Focused on structured scenarios, does not support additional features
* Simple and intuitive calling method. For comparison, here is a structured output example from go-openapi
```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

func main() {
	client := openai.NewClient("your token")
	ctx := context.Background()

	type Result struct {
		Steps []struct {
			Explanation string `json:"explanation"`
			Output      string `json:"output"`
		} `json:"steps"`
		FinalAnswer string `json:"final_answer"`
	}
	var result Result
	schema, err := jsonschema.GenerateSchemaForType(result)
	if err != nil {
		log.Fatalf("GenerateSchemaForType error: %v", err)
	}
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: "You are a helpful math tutor. Guide the user through the solution step by step.",
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: "how can I solve 8x + 7 = -23",
			},
		},
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:   "math_reasoning",
				Schema: schema,
				Strict: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("CreateChatCompletion error: %v", err)
	}
	err = schema.Unmarshal(resp.Choices[0].Message.Content, &result)
	if err != nil {
		log.Fatalf("Unmarshal schema error: %v", err)
	}
	fmt.Println(result)
}
```

## Best Practices

* Use the `desc` tag to describe field meanings
* Use the `enum` tag to describe field options

These tags are automatically injected into the generated JSON Schema to enrich the context

## License

MIT License