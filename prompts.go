package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// registerPrompts registers all prompts with the given server
func registerPrompts(s *server.MCPServer) {
	// Prompt for adjusting the semantic ratio in hybrid search
	adjustSemanticRatioPrompt := mcp.NewPrompt(
		"adjust_semantic_ratio",
		mcp.WithPromptDescription("Guide for adjusting the semantic_ratio in Meilisearch hybrid search."),
		mcp.WithArgument("search_type",
			mcp.ArgumentDescription("The type of search focus ('semantic' or 'keyword'). Defaults to 'balanced'."),
			// Argument is optional by default if RequiredArgument() is not called.
		),
	)
	s.AddPrompt(adjustSemanticRatioPrompt, handleAdjustSemanticRatioPrompt)

	// General help prompt for the search tool
	searchHelpPrompt := mcp.NewPrompt(
		"hybrid_search_help",
		mcp.WithPromptDescription("Guide on how to use the hybrid_search tool."),
	)
	s.AddPrompt(searchHelpPrompt, handleHybridSearchHelpPrompt)
}

// Handler for the semantic ratio adjustment prompt
func handleAdjustSemanticRatioPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	searchType := "balanced" // Default value
	// Explicitly check if the argument exists and then perform type assertion
	var searchTypeArg interface{}
	var ok bool
	searchTypeArg, ok = request.Params.Arguments["search_type"]
	if ok {
		st, isString := searchTypeArg.(string)
		if isString {
			searchType = st
		}
		// If the argument exists but is not a string, the default "balanced" will be used.
	}

	var ratioGuide string
	var examples string

	switch searchType {
	case "semantic":
		ratioGuide = "To prioritize semantic search, set semantic_ratio to 0.7 or higher."
		examples = "Example: For the keyword 'database design', documents related to 'data structures' or 'schema design' can also be found."
	case "keyword":
		ratioGuide = "To prioritize keyword search, set semantic_ratio to 0.3 or lower."
		examples = "Example: For the keyword 'Python', only documents containing the exact word 'Python' will be prioritized."
	default: // balanced
		ratioGuide = "For balanced results, set semantic_ratio around 0.5."
		examples = "Example: For the keyword 'machine learning', you'll get a mix of documents containing 'machine learning' and related topics like 'AI' or 'deep learning'."
	}

	return &mcp.GetPromptResult{
		Description: "Meilisearch Hybrid Search Parameter Guide",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Guide for adjusting the semantic_ratio parameter when using the hybrid_search tool:\n\n%s\n\n%s\n\nValue range: 0.0 (pure keyword) to 1.0 (pure semantic).", ratioGuide, examples),
				},
			},
			{
				Role: mcp.RoleAssistant,
				Content: mcp.TextContent{
					Type: "text",
					Text: "Understood. I will adjust the semantic_ratio based on the search goal to optimize hybrid search results.",
				},
			},
		},
	}, nil
}

// Handler for the search help prompt
func handleHybridSearchHelpPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	return &mcp.GetPromptResult{
		Description: "Basic usage guide for the hybrid_search tool",
		Messages: []mcp.PromptMessage{
			{
				Role: mcp.RoleUser,
				Content: mcp.TextContent{
					Type: "text",
					Text: `Basic usage of the hybrid_search tool:

1. Basic Search (balanced keyword and semantic):
			hybrid_search(keywords="your search terms")

2. Prioritize Semantic Search:
			hybrid_search(keywords="your search terms", semantic_ratio=0.8)

3. Prioritize Keyword Search:
			hybrid_search(keywords="your search terms", semantic_ratio=0.2)

4. Filtering Results (Optional):
			To filter results based on a specific attribute value, you **must provide both** 'filterable_attribute' and 'filter_word'.
			- 'filterable_attribute': The name of the attribute in your Meilisearch index that is configured as filterable (e.g., "genre", "author", "product_category").
			- 'filter_word': The specific value you want to filter by for the given attribute (e.g., "Drama", "Tolkien", "electronics").

			Syntax:
			hybrid_search(keywords="your search terms", filterable_attribute="attribute_name", filter_word="value_to_filter")

			Examples:
			- Find sci-fi movies:
					hybrid_search(keywords="movie about space", filterable_attribute="genre", filter_word="Science Fiction")
			- Find books by a specific author:
					hybrid_search(keywords="fantasy books", filterable_attribute="author", filter_word="Tolkien")
			- Find documents related to a specific product category:
					hybrid_search(keywords="latest gadgets", filterable_attribute="category", filter_word="Electronics")

5. For detailed guidance on semantic_ratio:
			Use: prompts/get(name="adjust_semantic_ratio", arguments={"search_type": "semantic"})
			(Replace "semantic" with "keyword" or omit for balanced guidance)`,
				},
			},
		},
	}, nil
}
