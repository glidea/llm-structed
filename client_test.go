package llmstructed

import (
	"context"
	"reflect"
	"testing"

	"github.com/pkg/errors"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:      "test-key",
				BaseURL:     "https://test.com",
				Model:       "test-model",
				Temperature: 0.5,
			},
			wantErr: false,
		},
		{
			name: "missing api key",
			config: Config{
				BaseURL:     "https://test.com",
				Model:       "test-model",
				Temperature: 0.5,
			},
			wantErr: true,
		},
		{
			name: "invalid temperature",
			config: Config{
				APIKey:      "test-key",
				BaseURL:     "https://test.com",
				Model:       "test-model",
				Temperature: 2.5,
			},
			wantErr: true,
		},
		{
			name: "default values",
			config: Config{
				APIKey: "test-key",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			method, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && method == nil {
				t.Error("New() returned nil method")
			}
		})
	}
}

func TestTypeToSchema(t *testing.T) {
	type Nested struct {
		Field string `json:"field_in_nested" desc:"nested field description"`
	}

	type TestStruct struct {
		StringField  string   `json:"string_field" desc:"string field description"`
		IntField     int      `json:"int_field"`
		FloatField   float64  `json:"float_field"`
		BoolField    bool     `json:"bool_field"`
		ArrayField   []string `json:"array_field"`
		NestedField  *Nested  `json:"nested_field"`
		IgnoreField  string   `json:"-"`
		privateField string
	}

	schema, err := typeToSchema(reflect.TypeOf(TestStruct{NestedField: &Nested{}}))
	if err != nil {
		t.Fatalf("typeToSchema() error = %v", err)
	}

	// Verify schema type
	if schema.Type != schemaTypeObject {
		t.Errorf("schema.Type = %v, want %v", schema.Type, schemaTypeObject)
	}

	// Verify properties
	expectedFields := map[string]schemaType{
		"string_field": schemaTypeString,
		"int_field":    schemaTypeInteger,
		"float_field":  schemaTypeNumber,
		"bool_field":   schemaTypeBoolean,
		"array_field":  schemaTypeArray,
		"nested_field": schemaTypeObject,
	}

	for field, expectedType := range expectedFields {
		prop, ok := schema.ObjectProperties[field]
		if !ok {
			t.Errorf("missing field %s in schema", field)
			continue
		}
		if prop.Type != expectedType {
			t.Errorf("field %s type = %v, want %v", field, prop.Type, expectedType)
		}
	}

	// Verify description
	if desc := schema.ObjectProperties["string_field"].Description; desc != "string field description" {
		t.Errorf("string_field description = %v, want 'string field description'", desc)
	}

	// Verify nested field
	nestedProp, ok := schema.ObjectProperties["nested_field"]
	if !ok {
		t.Error("missing nested_field in schema")
	} else {
		nestedField, ok := nestedProp.ObjectProperties["field_in_nested"]
		if !ok {
			t.Error("missing field_in_nested in nested schema")
		} else {
			if nestedField.Type != schemaTypeString {
				t.Errorf("nested field type = %v, want %v", nestedField.Type, schemaTypeString)
			}
			if nestedField.Description != "nested field description" {
				t.Errorf("nested field description = %v, want 'nested field description'", nestedField.Description)
			}
		}
	}

	// Verify ignored fields
	if _, ok := schema.ObjectProperties["IgnoreField"]; ok {
		t.Error("IgnoreField should be ignored")
	}
	if _, ok := schema.ObjectProperties["privateField"]; ok {
		t.Error("privateField should be ignored")
	}
}

func TestDo(t *testing.T) {
	type TestResponse struct {
		Message string `json:"message"`
	}

	tests := []struct {
		name      string
		responses [][]byte
		errors    []error
		retry     int
		want      TestResponse
		wantErr   bool
	}{
		{
			name: "successful call",
			responses: [][]byte{
				[]byte(`{"message":"success"}`),
			},
			errors:  []error{nil},
			retry:   0,
			want:    TestResponse{Message: "success"},
			wantErr: false,
		},
		{
			name: "retry success",
			responses: [][]byte{
				nil,
				[]byte(`{"message":"retry success"}`),
			},
			errors: []error{
				errors.New("first attempt failed"),
				nil,
			},
			retry:   1,
			want:    TestResponse{Message: "retry success"},
			wantErr: false,
		},
		{
			name: "all attempts fail",
			responses: [][]byte{
				nil,
				nil,
			},
			errors: []error{
				errors.New("first attempt failed"),
				errors.New("second attempt failed"),
			},
			retry:   1,
			want:    TestResponse{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: tt.responses,
				errors:    tt.errors,
			}

			c := &client{
				llm:   mockLLM,
				retry: tt.retry,
			}

			var got TestResponse
			if err := c.Do(context.Background(), []string{"test message"}, &got); (err != nil) != tt.wantErr {
				t.Errorf("Do() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Do() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestString(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    string
		wantErr bool
	}{
		{
			name:    "successful string response",
			mock:    `{"value": "string"}`,
			want:    "string",
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"value": 123}`,
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.String(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("String() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringSlice(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    []string
		wantErr bool
	}{
		{
			name:    "successful string slice response",
			mock:    `{"values":["value1","value2"]}`,
			want:    []string{"value1", "value2"},
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"values": "abc"}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.StringSlice(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("StringSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBool(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    bool
		wantErr bool
	}{
		{
			name:    "successful bool response",
			mock:    `{"value":true}`,
			want:    true,
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"value":"false"}`,
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.Bool(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Bool() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Bool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoolSlice(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    []bool
		wantErr bool
	}{
		{
			name:    "successful bool slice response",
			mock:    `{"values":[true,false]}`,
			want:    []bool{true, false},
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"values": ["true"]}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.BoolSlice(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("BoolSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BoolSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    int
		wantErr bool
	}{
		{
			name:    "successful int response",
			mock:    `{"value":42}`,
			want:    42,
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"value":"42"}`,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.Int(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Int() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Int() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIntSlice(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    []int
		wantErr bool
	}{
		{
			name:    "successful int slice response",
			mock:    `{"values":[1,2,3]}`,
			want:    []int{1, 2, 3},
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"values": ["1"]}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.IntSlice(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("IntSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("IntSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloat(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    float32
		wantErr bool
	}{
		{
			name:    "successful float response",
			mock:    `{"value":3.14}`,
			want:    3.14,
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"value":"2.12"}`,
			want:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.Float(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("Float() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Float() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFloatSlice(t *testing.T) {
	tests := []struct {
		name    string
		mock    string
		want    []float32
		wantErr bool
	}{
		{
			name:    "successful float slice response",
			mock:    `{"values":[1.1,2.2,3.3]}`,
			want:    []float32{1.1, 2.2, 3.3},
			wantErr: false,
		},
		{
			name:    "error response",
			mock:    `{"values": ["223.1"]}`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLLM := &mockLLM{
				responses: [][]byte{
					[]byte(tt.mock),
				},
				errors: []error{nil},
			}

			c := &client{
				llm: mockLLM,
			}

			got, err := c.FloatSlice(context.Background(), []string{})
			if (err != nil) != tt.wantErr {
				t.Errorf("FloatSlice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FloatSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}
