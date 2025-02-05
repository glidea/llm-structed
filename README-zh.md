

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

## 最佳实践

* 使用 `desc` 标签来描述字段含义
* 使用 `enum` 标签来描述字段可选值

这些标签会自动注入到生成的 JSON Schema 中，以丰富上下文

## 许可证

MIT License
