[中文](README-zh.md)

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
    "github.com/glidea/llm-structed"
)

type Response struct {
    Title    string   `desc:"The title of the summary"`
    Content  string   `desc:"A concise summary of the article content"`
    Keywords []string `desc:"Key topics mentioned in the article"`
    Score    int      `desc:"The quality score of the article (1-10)"`
    Category string   `desc:"The category of the article" enum:"Technology,Science,Business,Health,Education,Other"`
}

func main() {
    client, err := llmstructed.New(llmstructed.Config{
        BaseURL: "https://api.openai.com/v1",
        APIKey:  "your-api-key",
        Model:   "gpt-4o-mini",
		StructuredOutputSupported: true,
    })

    var resp Response
    err = client.Do(context.Background(), []string{
        "Please generate a summary of this article: Artificial Intelligence (AI) is transforming the way we live and work...",
    }, &resp)
}

// Equals to
curl -X POST https://api.openai.com/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-api-key' \
  -d '{"messages":[{"content":"You are a helpful assistant that provides structured output. Your response must be a valid JSON object.","role":"system"},{"content":"Please generate a summary of this article: Artificial Intelligence (AI) is transforming the way we live and work...","role":"user"}],"model":"gpt-4o-mini","provider":{"require_parameters":true},"response_format":{"json_schema":{"name":"response","schema":{"additionalProperties":false,"properties":{"category":{"description":"The category of the article","enum":["Technology","Science","Business","Health","Education","Other"],"type":"string"},"content":{"description":"A concise summary of the article content","type":"string"},"keywords":{"description":"Key topics mentioned in the article","items":{"type":"string"},"type":"array"},"score":{"description":"The quality score of the article (1-10)","type":"integer"},"title":{"description":"The title of the summary","type":"string"}},"required":["category","title","content","keywords","score"],"type":"object"},"strict":true},"type":"json_schema"},"temperature":0}'
```
Complete example and explanation: [example/example.go](example/example.go)

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