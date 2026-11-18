package ai

type GenerateBlogRequest struct {
	Topic            string `json:"topic" binding:"required"`
	Category         string `json:"category"`
	IncludeWebSearch bool   `json:"include_web_search"`
}

// GeneratedPost adalah draf artikel utuh hasil AI beserta daftar sumber rujukan.
type GeneratedPost struct {
	Title    string   `json:"title"`
	Category string   `json:"category"`
	Excerpt  string   `json:"excerpt"`
	Content  string   `json:"content"`
	Sources  []string `json:"sources"`
}

type SearchResultItem struct {
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
	URL     string `json:"url"`
}

type SearchEngineResponse struct {
	Results []SearchResultItem `json:"results"`
}
