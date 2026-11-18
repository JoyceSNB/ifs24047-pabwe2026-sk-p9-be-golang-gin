package ai

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

type mockAiClient struct {
	post     *GeneratedPost
	err      error
	checkErr error
}

func (m *mockAiClient) CheckAccess() error {
	return m.checkErr
}

func (m *mockAiClient) ChatCompletion(topic, category string, searchSnippets []string) (*GeneratedPost, error) {
	if m.err != nil {
		return nil, m.err
	}
	cp := *m.post
	return &cp, nil
}

type mockSearchEngineClient struct {
	results []string
	err     error
}

func (m *mockSearchEngineClient) Search(query string) ([]string, error) {
	return m.results, m.err
}

func newAiCtx(body []byte) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest(http.MethodPost, "/api/posts/ai-generate", bytes.NewBuffer(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

func TestAiService_And_Handler(t *testing.T) {
	mockClient := &mockAiClient{
		post: &GeneratedPost{
			Title:    "Belajar Gin",
			Category: "teknologi",
			Excerpt:  "Rute dan Middleware",
			Content:  "## Gin\n\nIsi artikel",
		},
	}
	mockSearch := &mockSearchEngineClient{
		results: []string{"Delcom - Belajar Go Gin (https://delcom.org)"},
	}

	service := NewAiService(mockClient, mockSearch)
	handler := NewAiHandler(service)

	// Success Handler without search
	req := GenerateBlogRequest{Topic: "Belajar Fullstack", IncludeWebSearch: false}
	bReq, _ := json.Marshal(req)
	c1, w1 := newAiCtx(bReq)
	handler.GenerateBlog(c1)
	assert.Equal(t, http.StatusOK, w1.Code)
	assert.Contains(t, w1.Body.String(), "Teknologi")

	// Success with search enabled: sumber diambil dari hasil penelusuran
	reqSearch := GenerateBlogRequest{Topic: "Belajar Fullstack", IncludeWebSearch: true}
	bReqSearch, _ := json.Marshal(reqSearch)
	cSearch, wSearch := newAiCtx(bReqSearch)
	handler.GenerateBlog(cSearch)
	assert.Equal(t, http.StatusOK, wSearch.Code)

	res, err := service.GenerateBlog(reqSearch)
	assert.NoError(t, err)
	assert.Equal(t, mockSearch.results, res.Sources)
	assert.Equal(t, "Teknologi", res.Category)

	// Sumber dari model dipertahankan
	mockClient.post.Sources = []string{"https://model.example.com"}
	res, err = service.GenerateBlog(reqSearch)
	assert.NoError(t, err)
	assert.Equal(t, []string{"https://model.example.com"}, res.Sources)
	mockClient.post.Sources = nil

	// Search returns error fallback
	mockSearch.err = errors.New("search engine down")
	resSearchErr, err := service.GenerateBlog(reqSearch)
	assert.NoError(t, err)
	assert.Empty(t, resSearchErr.Sources)
	mockSearch.err = nil

	// Search returns empty results
	mockSearch.results = nil
	resEmptySearch, err := service.GenerateBlog(reqSearch)
	assert.NoError(t, err)
	assert.Empty(t, resEmptySearch.Sources)
	mockSearch.results = []string{"Delcom - Belajar Go Gin (https://delcom.org)"}

	// Web search diminta tetapi search client nil
	serviceNoSearch := NewAiService(mockClient, nil)
	resNoSearch, err := serviceNoSearch.GenerateBlog(reqSearch)
	assert.NoError(t, err)
	assert.Empty(t, resNoSearch.Sources)

	// Kategori dari model tidak valid -> pakai kategori permintaan
	mockClient.post.Category = "Kategori Aneh"
	resCat, err := service.GenerateBlog(GenerateBlogRequest{Topic: "Topik", Category: "bisnis"})
	assert.NoError(t, err)
	assert.Equal(t, "Bisnis", resCat.Category)

	// Kategori model dan permintaan sama-sama tidak valid -> Umum
	resCatDefault, err := service.GenerateBlog(GenerateBlogRequest{Topic: "Topik", Category: "Salah"})
	assert.NoError(t, err)
	assert.Equal(t, "Umum", resCatDefault.Category)
	mockClient.post.Category = "teknologi"

	// Invalid JSON
	c2, w2 := newAiCtx([]byte("{invalid"))
	handler.GenerateBlog(c2)
	assert.Equal(t, http.StatusBadRequest, w2.Code)

	// Empty Topic in service
	_, err = service.GenerateBlog(GenerateBlogRequest{Topic: ""})
	assert.Error(t, err)
	_, err = service.GenerateBlog(GenerateBlogRequest{Topic: "   "})
	assert.Error(t, err)

	// OpenAI check error before web search
	mockClient.checkErr = errors.New("cannot reach openai")
	_, err = service.GenerateBlog(req)
	assert.Error(t, err)
	mockClient.checkErr = nil

	// Client error in service / handler
	mockClient.err = errors.New("upstream timeout")
	c3, w3 := newAiCtx(bReq)
	handler.GenerateBlog(c3)
	assert.Equal(t, http.StatusInternalServerError, w3.Code)
}

func TestOpenAiClient_ChatCompletion(t *testing.T) {
	// 1. Success with raw JSON and searchSnippets
	serverOk := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		if r.URL.Path == "/models" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[]}`))
			return
		}
		resp := openAiResponse{
			Choices: []openAiChoice{
				{
					Message: openAiMessage{
						Role:    "assistant",
						Content: `{"title":"Judul 1","category":"Teknologi","excerpt":"Ringkasan","content":"## Isi","sources":["https://example.com"]}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverOk.Close()

	client := NewOpenAiClient(serverOk.URL, "test-key", "test-model")
	post, err := client.ChatCompletion("topik blog", "Teknologi", []string{"Snippet 1", "Snippet 2"})
	assert.NoError(t, err)
	assert.Equal(t, "Judul 1", post.Title)
	assert.Equal(t, []string{"https://example.com"}, post.Sources)

	// CheckAccess tests
	assert.NoError(t, client.CheckAccess())
	clientNoKey := NewOpenAiClient(serverOk.URL, "", "test-model")
	assert.Error(t, clientNoKey.CheckAccess())
	clientBadUrl := NewOpenAiClient("http://127.0.0.1:59999", "test-key", "test-model")
	assert.Error(t, clientBadUrl.CheckAccess())

	serverUnauthorized := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
	}))
	defer serverUnauthorized.Close()
	clientUnauthorized := NewOpenAiClient(serverUnauthorized.URL, "bad-key", "test-model")
	assert.Error(t, clientUnauthorized.CheckAccess())

	// 2. Success with markdown fences and no searchSnippets
	serverMarkdown := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAiResponse{
			Choices: []openAiChoice{
				{
					Message: openAiMessage{
						Role: "assistant",
						Content: "```json\n" +
							`{"title":"Judul Markdown","category":"Umum","excerpt":"Ringkasan","content":"Isi"}` +
							"\n```",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverMarkdown.Close()

	clientMd := NewOpenAiClient(serverMarkdown.URL, "test-key", "test-model")
	postMd, err := clientMd.ChatCompletion("topik blog", "", nil)
	assert.NoError(t, err)
	assert.Equal(t, "Judul Markdown", postMd.Title)

	// 3. Upstream HTTP Error (status 500)
	serverErr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal server error"))
	}))
	defer serverErr.Close()

	clientErr := NewOpenAiClient(serverErr.URL, "test-key", "test-model")
	_, err = clientErr.ChatCompletion("topik blog", "Umum", nil)
	assert.Error(t, err)

	// 4. Upstream Empty choices
	serverEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAiResponse{Choices: []openAiChoice{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverEmpty.Close()

	clientEmpty := NewOpenAiClient(serverEmpty.URL, "test-key", "test-model")
	_, err = clientEmpty.ChatCompletion("topik blog", "Umum", nil)
	assert.Error(t, err)

	// 5. Invalid JSON response from AI
	serverBadJson := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAiResponse{
			Choices: []openAiChoice{
				{
					Message: openAiMessage{
						Role:    "assistant",
						Content: `Not a json string at all`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverBadJson.Close()

	clientBadJson := NewOpenAiClient(serverBadJson.URL, "test-key", "test-model")
	_, err = clientBadJson.ChatCompletion("topik blog", "Umum", nil)
	assert.Error(t, err)

	// 6. Network failure / invalid URL
	clientNetFail := NewOpenAiClient("http://127.0.0.1:54399", "test-key", "test-model")
	_, err = clientNetFail.ChatCompletion("topik blog", "Umum", nil)
	assert.Error(t, err)

	// 7. Malformed unmarshal response
	serverGarbage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html>error</html>"))
	}))
	defer serverGarbage.Close()

	clientGarbage := NewOpenAiClient(serverGarbage.URL, "test-key", "test-model")
	_, err = clientGarbage.ChatCompletion("topik blog", "Umum", nil)
	assert.Error(t, err)
}

func TestSearchEngineClient(t *testing.T) {
	// Default client construction
	defClient := NewSearchEngineClient("")
	assert.NotNil(t, defClient)

	// Empty query returns empty
	emptyRes, err := defClient.Search("   ")
	assert.NoError(t, err)
	assert.Empty(t, emptyRes)

	// 1. Wikipedia OpenSearch parsing [term, titles, descriptions, urls]
	serverWikiOS := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := `["golang", ["Golang", "Go Lang No Desc", "Only Title"], ["Bahasa pemrograman modern", "", ""], ["https://id.wikipedia.org/wiki/Golang", "https://id.wikipedia.org/wiki/Go", ""]]`
		_, _ = w.Write([]byte(raw))
	}))
	defer serverWikiOS.Close()

	clientWikiOS := NewSearchEngineClient(serverWikiOS.URL+"/wikipedia.org/api.php?custom=1", serverWikiOS.Client())
	resWikiOS, err := clientWikiOS.Search("golang")
	assert.NoError(t, err)
	assert.Len(t, resWikiOS, 3)
	assert.Contains(t, resWikiOS[0], "Bahasa pemrograman modern")
	assert.Contains(t, resWikiOS[1], "https://id.wikipedia.org/wiki/Go")
	assert.Equal(t, "Only Title", resWikiOS[2])

	// 2. Wikipedia Query parsing {"query": {"search": [...]}}
	serverWikiQuery := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := `{"query":{"search":[{"title":"React","snippet":"React is a <span>JS</span> library &amp; &quot;UI&quot; framework"}]}}`
		_, _ = w.Write([]byte(raw))
	}))
	defer serverWikiQuery.Close()

	clientWikiQuery := NewSearchEngineClient(serverWikiQuery.URL+"/wikipedia.org", serverWikiQuery.Client())
	resWikiQuery, err := clientWikiQuery.Search("React")
	assert.NoError(t, err)
	assert.Len(t, resWikiQuery, 1)
	assert.Contains(t, resWikiQuery[0], "React is a JS library & \"UI\" framework")

	// 3. DuckDuckGo parsing
	serverDDG := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := `{"Heading":"Go","AbstractText":"Open source programming language","AbstractURL":"https://duckduckgo.com/Go","RelatedTopics":[{"Text":"Go syntax","FirstURL":"https://duckduckgo.com/syntax"}]}`
		_, _ = w.Write([]byte(raw))
	}))
	defer serverDDG.Close()

	clientDDG := NewSearchEngineClient(serverDDG.URL+"/duckduckgo.com?lang=id", serverDDG.Client())
	resDDG, err := clientDDG.Search("Go")
	assert.NoError(t, err)
	assert.Len(t, resDDG, 2)
	assert.Contains(t, resDDG[0], "Open source programming language")

	// 4. Standard SearXNG / Delcom format with empty snippet and valid snippet
	serverStandard := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SearchEngineResponse{
			Results: []SearchResultItem{
				{Title: "Title 1", Snippet: "Snippet 1", URL: "https://example.com/1"},
				{Title: "Title 2", Snippet: "", URL: "https://example.com/2"},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverStandard.Close()

	clientStandard := NewSearchEngineClient(serverStandard.URL+"?lang=en", serverStandard.Client())
	resStandard, err := clientStandard.Search("test")
	assert.NoError(t, err)
	assert.Len(t, resStandard, 2)
	assert.Contains(t, resStandard[0], "Snippet 1")
	assert.Contains(t, resStandard[1], "(https://example.com/2)")

	// 5. Server OK but empty results -> fallback information
	serverEmpty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SearchEngineResponse{Results: []SearchResultItem{}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer serverEmpty.Close()

	clientEmpty := NewSearchEngineClient(serverEmpty.URL, serverEmpty.Client())
	resEmpty, err := clientEmpty.Search("unknown query")
	assert.NoError(t, err)
	assert.Len(t, resEmpty, 1)
	assert.Contains(t, resEmpty[0], "Informasi relevan mengenai: unknown query")

	// 6. Server Plain text body fallback
	serverNonJson := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("Plain text search snippet &amp; notes"))
	}))
	defer serverNonJson.Close()

	clientNonJson := NewSearchEngineClient(serverNonJson.URL, serverNonJson.Client())
	resNonJson, err := clientNonJson.Search("plain")
	assert.NoError(t, err)
	assert.Equal(t, []string{"Plain text search snippet & notes"}, resNonJson)

	// 7. Server Status != 200
	serverError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer serverError.Close()

	clientError := NewSearchEngineClient(serverError.URL, serverError.Client())
	_, err = clientError.Search("error query")
	assert.Error(t, err)

	// 8. Invalid URL / Request creation error (using invalid URL character)
	clientBadURL := NewSearchEngineClient("http://[fe80::1%en0]:invalid-url", &http.Client{})
	_, err = clientBadURL.Search("bad")
	assert.Error(t, err)

	// 9. Network connection failure
	clientNetFail := NewSearchEngineClient("http://127.0.0.1:54398", &http.Client{})
	_, err = clientNetFail.Search("fail")
	assert.Error(t, err)
}
