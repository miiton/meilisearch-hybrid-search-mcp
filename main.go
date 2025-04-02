package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/meilisearch/meilisearch-go"
)

func main() {
	s := server.NewMCPServer(
		"Hybrid search your documents in Meilisearch",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

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

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
