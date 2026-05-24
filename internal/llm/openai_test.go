package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOpenAISummarizeUsesResponsesAPI(t *testing.T) {
	var gotPath, gotAuth string
	var gotBody openAIResponsesRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output_text":"summary"}`))
	}))
	defer server.Close()

	client := NewOpenAI("test-key", "gpt-test", server.URL+"/", "responses")
	client.httpClient = server.Client()

	got, err := client.Summarize(nil, SummarizeOpts{
		Language: "en",
		Username: "octocat",
		Start:    time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if got != "summary" {
		t.Fatalf("summary = %q, want %q", got, "summary")
	}
	if gotPath != "/responses" {
		t.Fatalf("path = %q, want /responses", gotPath)
	}
	if gotAuth != "Bearer test-key" {
		t.Fatalf("auth = %q, want bearer token", gotAuth)
	}
	if gotBody.Model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", gotBody.Model)
	}
	if gotBody.Input == "" {
		t.Fatalf("input prompt is empty")
	}
}

func TestOpenAISummarizeUsesChatCompletionsAPI(t *testing.T) {
	var gotPath string
	var gotBody openAIChatRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"chat summary"}}]}`))
	}))
	defer server.Close()

	client := NewOpenAI("test-key", "gpt-test", server.URL, "chat")
	client.httpClient = server.Client()

	got, err := client.Summarize(nil, SummarizeOpts{
		Language: "en",
		Username: "octocat",
		Start:    time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC),
		End:      time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Summarize returned error: %v", err)
	}
	if got != "chat summary" {
		t.Fatalf("summary = %q, want %q", got, "chat summary")
	}
	if gotPath != "/chat/completions" {
		t.Fatalf("path = %q, want /chat/completions", gotPath)
	}
	if gotBody.Model != "gpt-test" {
		t.Fatalf("model = %q, want gpt-test", gotBody.Model)
	}
	if len(gotBody.Messages) != 1 || gotBody.Messages[0].Role != "user" || gotBody.Messages[0].Content == "" {
		t.Fatalf("messages = %#v, want one user prompt", gotBody.Messages)
	}
}

func TestOpenAIResponsesResponseTextFromOutputContent(t *testing.T) {
	resp := openAIResponsesResponse{
		Output: []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		}{
			{
				Content: []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				}{
					{Type: "output_text", Text: "first"},
					{Type: "output_text", Text: "second"},
				},
			},
		},
	}

	got := resp.Text()
	if got != "first\nsecond" {
		t.Fatalf("Text() = %q, want joined content text", got)
	}
}
