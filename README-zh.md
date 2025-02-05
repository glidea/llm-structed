

# llm-structed

llm-structed 是一个针对结构化输出场景优化的 LLM Client：
* 自动将 Go 结构体定义转换为 Response JSON Schema
* 自动把 LLM 的输出转换为 Go 结构体
* 基于 struct tags 的友好声明式配置
* 轻量
* 基于 [Json Schema or Json Object](https://platform.openai.com/docs/guides/structured-outputs#supported-schemas)
* 只支持 OpenAI 兼容的 LLM，主流提供商基本都有对应的兼容接口，比如 [Gemini](https://ai.google.dev/gemini-api/docs/openai)

## 安装

```bash
go get github.com/glidea/llm-structed
```

## 快速开始
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
    })

    var resp Response
    err = client.Do(context.Background(), []string{
        "Please generate a summary of this article: Artificial Intelligence (AI) is transforming the way we live and work...",
    }, &resp)
}
```
完整示例与说明：[example/example.go](example/example.go)

## 与 [go-openapi](https://github.com/sashabaranov/go-openai) 的区别

* 更轻量
* 专注与结构化场景，不支持额外功能
* 调用方式简单直观。作为对比以下是 go-openapi 的结构化输出示例
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

// Equals to
curl -X POST https://api.openai.com/v1/chat/completions \
  -H 'Content-Type: application/json' \
  -H 'Authorization: Bearer your-api-key' \
  -d '{"messages":[{"content":"You are a helpful assistant that provides structured output. Your response must be a valid JSON object.","role":"system"},{"content":"Please generate a summary of this article: Artificial Intelligence (AI) is transforming the way we live and work...","role":"user"}],"model":"gpt-4o-mini","provider":{"require_parameters":true},"response_format":{"json_schema":{"name":"response","schema":{"additionalProperties":false,"properties":{"category":{"description":"The category of the article","enum":["Technology","Science","Business","Health","Education","Other"],"type":"string"},"content":{"description":"A concise summary of the article content","type":"string"},"keywords":{"description":"Key topics mentioned in the article","items":{"type":"string"},"type":"array"},"score":{"description":"The quality score of the article (1-10)","type":"integer"},"title":{"description":"The title of the summary","type":"string"}},"required":["category","title","content","keywords","score"],"type":"object"},"strict":true},"type":"json_schema"},"temperature":0}'
```

## 最佳实践

* 使用 `desc` 标签来描述字段含义
* 使用 `enum` 标签来描述字段可选值

这些标签会自动注入到生成的 JSON Schema 中，以丰富上下文

## 许可证

MIT License
