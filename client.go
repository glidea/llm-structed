package llmstructed

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"sync"

	"github.com/pkg/errors"
)

type Client interface {
	Do(ctx context.Context, messages []string, ret any) error
}

// Config contains the configuration options for the LLM client.
// Only OpenAI compatible models are supported.
type Config struct {
	// Debug is used to print debug info for curl the final request.
	// WARNING: your API key will be printed in the request, so don't set it to true in production environment.
	// Default: false
	Debug bool
	// BaseURL is the base URL of the endpoint
	// Default: https://api.deepseek.com/v1
	BaseURL string
	// APIKey is the authentication key
	APIKey string
	// Model specifies which model to use
	// Default: deepseek-chat
	Model string
	// Temperature controls randomness in the model's output (0.0-2.0)
	// Recommended to use lower values for stable structured output, especially when Model doesn't support structured output
	// Default: 0.0
	Temperature float32
	// StructuredOutputSupported indicates whether the model supports structured output,
	// else the output structure is not guaranteed, especially for some low-quality models.
	// But if you not sure, MUST set it to false.
	// See https://platform.openai.com/docs/guides/structured-outputs
	// Default: false
	StructuredOutputSupported bool
	// Retry specifies how many times to retry failed requests.
	// When StructuredOutputSupported=false, it's recommended to enable retry.
	// Default: 1
	Retry int
}

type client struct {
	llm         llm
	retry       int
	schemaCache sync.Map
}

func New(config Config) (Client, error) {
	if config.APIKey == "" {
		return nil, errors.New("api key is required")
	}
	if config.Temperature < 0 || config.Temperature > 2 {
		return nil, errors.New("temperature must be between 0 and 2")
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1"
	}
	if config.Model == "" {
		config.Model = "deepseek-chat"
	}

	llm := &openai{
		config: llmConfig{
			Debug:                     config.Debug,
			BaseURL:                   config.BaseURL,
			APIKey:                    config.APIKey,
			Model:                     config.Model,
			Temperature:               config.Temperature,
			StructuredOutputSupported: config.StructuredOutputSupported,
		},
		hc: &http.Client{},
	}

	return &client{
		llm:   llm,
		retry: config.Retry,
	}, nil
}

func typeToSchema(t reflect.Type) (*schema, error) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	switch t.Kind() {
	case reflect.String:
		return &schema{Type: schemaTypeString}, nil
	case reflect.Float32, reflect.Float64:
		return &schema{Type: schemaTypeNumber}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &schema{Type: schemaTypeInteger}, nil
	case reflect.Bool:
		return &schema{Type: schemaTypeBoolean}, nil
	case reflect.Slice, reflect.Array:
		s, err := typeToSchema(t.Elem())
		if err != nil {
			return nil, err
		}
		return &schema{
			Type:       schemaTypeArray,
			ArrayItems: s,
		}, nil
	case reflect.Struct:
		properties := make(map[string]*schema)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" {
				continue
			}
			jsonTag := field.Tag.Get("json")
			if jsonTag == "-" {
				continue
			}

			name := field.Name
			if jsonTag != "" {
				if comma := strings.Index(jsonTag, ","); comma != -1 {
					name = jsonTag[:comma]
				} else {
					name = jsonTag
				}
			}
			s, err := typeToSchema(field.Type)
			if err != nil {
				return nil, err
			}
			s.Description = field.Tag.Get("desc")
			if s.Type == schemaTypeString {
				if enumTag := field.Tag.Get("enum"); enumTag != "" {
					s.Enum = strings.Split(enumTag, ",")
				}
			}
			properties[name] = s
		}
		return &schema{
			Type:             schemaTypeObject,
			ObjectProperties: properties,
		}, nil
	default:
		return nil, errors.Errorf("unsupported type: %s", t.Kind())
	}
}

func (c *client) Do(ctx context.Context, messages []string, ret any) error {
	v := reflect.ValueOf(ret)
	if v.Kind() != reflect.Ptr {
		return errors.New("ret must be a pointer")
	}

	t := v.Elem().Type()
	if t.Kind() != reflect.Struct {
		return errors.Errorf("ret must be a pointer to struct, got %s", t.Kind())
	}

	var sche *schema
	if cached, ok := c.schemaCache.Load(t); ok {
		sche = cached.(*schema)
	} else {
		schema, err := typeToSchema(t)
		if err != nil {
			return err
		}
		sche = schema
		c.schemaCache.Store(t, schema)
	}

	var lastErr error
	retries := c.retry
	if retries <= 0 {
		retries = 1
	}

	for i := 0; i < retries+1; i++ {
		respBytes, err := c.llm.Completions(ctx, messages, sche)
		if err != nil {
			lastErr = err
			continue
		}

		if err := json.Unmarshal(respBytes, ret); err != nil {
			lastErr = errors.Wrapf(err, "unmarshal response: %s", string(respBytes))
			continue
		}

		return nil
	}

	return lastErr
}
