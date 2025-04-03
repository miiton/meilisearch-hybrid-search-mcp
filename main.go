package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings" // Add strings package

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/meilisearch/meilisearch-go"
)

func main() {
	s := server.NewMCPServer(
		"Meilisearch Hybrid Search MCP", // Updated description to English
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)
	// Read filterable attributes from environment variable
	filterableAttrsStr := os.Getenv("MEILI_FILTERABLE_ATTRIBUTES")
	filterableAttrs := []string{}
	if filterableAttrsStr != "" {
		filterableAttrs = strings.Split(filterableAttrsStr, ",")
		// Trim whitespace from each attribute
		for i := range filterableAttrs {
			filterableAttrs[i] = strings.TrimSpace(filterableAttrs[i])
		}
	}

	// Generate description for filterable_attribute argument
	filterableAttrDescription := "Attribute to filter on. Requires filter_word."
	if len(filterableAttrs) > 0 {
		// Format the list nicely for the description
		availableAttrsStr := strings.Join(filterableAttrs, ", ")
		filterableAttrDescription = fmt.Sprintf("Attribute to filter on (Available: %s). Requires filter_word.", availableAttrsStr)
	}

	searchTool := mcp.NewTool("hybrid_search",
		mcp.WithDescription("Hybrid search your documents in Meilisearch index"),
		mcp.WithString("keywords",
			mcp.Required(),
			mcp.Description("Keywords to search for"),
		),
		mcp.WithNumber("semantic_ratio",
			mcp.Description("Semantic ratio"),
			mcp.DefaultNumber(0.5),
			mcp.Min(0.0),
			mcp.Max(1.0),
		),
		// Add optional filtering arguments
		mcp.WithString("filterable_attribute",
			mcp.Description(filterableAttrDescription), // Use dynamically generated description
			// Argument is optional by default
		),
		mcp.WithString("filter_word",
			mcp.Description("Word or value to filter the attribute by (e.g., 'Drama', 'Tolkien'). Requires filterable_attribute."),
			// Argument is optional by default
		),
	)

	s.AddTool(searchTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		keywordsValue, ok := request.Params.Arguments["keywords"]
		if !ok {
			return nil, errors.New("missing required argument: keywords")
		}
		keywords, ok := keywordsValue.(string)
		if !ok {
			return nil, errors.New("argument 'keywords' must be a string")
		}

		semanticRatioValue, ok := request.Params.Arguments["semantic_ratio"]
		if !ok {
			semanticRatioValue = 0.5
		}
		semanticRatio, ok := semanticRatioValue.(float64)
		if !ok {
			return nil, errors.New("argument 'semantic_ratio' must be a number")
		}

		// Handle optional filtering arguments
		var filterAttribute, filterWord string
		var filterExpr string

		filterAttrValue, filterAttrOk := request.Params.Arguments["filterable_attribute"]
		filterWordValue, filterWordOk := request.Params.Arguments["filter_word"]

		if filterAttrOk && filterWordOk {
			var attrOk, wordOk bool
			filterAttribute, attrOk = filterAttrValue.(string)
			filterWord, wordOk = filterWordValue.(string)

			if attrOk && wordOk && filterAttribute != "" && filterWord != "" {
				// Construct the filter string: attribute = 'word'
				// Ensure proper quoting for the filter word
				filterExpr = fmt.Sprintf(`%s = '%s'`, filterAttribute, filterWord)
			}
		}
		// Removed extra closing brace here

		meiliHost := os.Getenv("MEILI_HOST")
		if meiliHost == "" {
			return nil, errors.New("environment variable MEILI_HOST is not set")
		}
		meiliAPIKey := os.Getenv("MEILI_API_KEY")
		meiliIndex := os.Getenv("MEILI_INDEX")
		if meiliIndex == "" {
			return nil, errors.New("environment variable MEILI_INDEX is not set")
		}
		meiliEmbedder := os.Getenv("MEILI_EMBEDDER")
		if meiliEmbedder == "" {
			return nil, errors.New("environment variable MEILI_EMBEDDER is not set")
		}

		client := meilisearch.New(meiliHost, meilisearch.WithAPIKey(meiliAPIKey))

		index := client.Index(meiliIndex)

		searchReq := &meilisearch.SearchRequest{
			Query: keywords,
			Hybrid: &meilisearch.SearchRequestHybrid{
				SemanticRatio: semanticRatio,
				Embedder:      meiliEmbedder,
			},
			Filter: filterExpr, // Add the constructed filter expression (will be empty if no valid filter args)
		}

		searchRes, err := index.Search(keywords, searchReq)
		if err != nil {
			return nil, fmt.Errorf("meilisearch search failed: %w", err)
		}

		jsonResult, err := json.Marshal(searchRes)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search result to JSON: %w", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	// Register prompts before starting the server
	registerPrompts(s)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
