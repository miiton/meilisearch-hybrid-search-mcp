package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/meilisearch/meilisearch-go"
)

var (
	hostFlag     *string
	apiKeyFlag   *string
	indexFlag    *string
	embedderFlag *string
)

func init() {
	hostFlag = flag.String("host", "", "Meilisearch server host (e.g., http://localhost:7700)")
	apiKeyFlag = flag.String("api-key", "", "Meilisearch API key")
	indexFlag = flag.String("index", "", "Meilisearch index name")
	embedderFlag = flag.String("embedder", "", "Embedder to use (e.g., openai)")
	flag.Parse()
}

func newMeiliIndex() meilisearch.IndexManager {
	meiliHost := *hostFlag
	if meiliHost == "" {
		meiliHost = os.Getenv("MEILI_HOST")
		if meiliHost == "" {
			fmt.Println("Error: Meilisearch host not provided. Use --host flag or set MEILI_HOST environment variable")
			os.Exit(1)
		}
	}

	meiliAPIKey := *apiKeyFlag
	if meiliAPIKey == "" {
		meiliAPIKey = os.Getenv("MEILI_API_KEY")
	}

	meiliIndex := *indexFlag
	if meiliIndex == "" {
		meiliIndex = os.Getenv("MEILI_INDEX")
		if meiliIndex == "" {
			fmt.Println("Error: Meilisearch index not provided. Use --index flag or set MEILI_INDEX environment variable")
			os.Exit(1)
		}
	}

	client := meilisearch.New(meiliHost, meilisearch.WithAPIKey(meiliAPIKey))
	index := client.Index(meiliIndex)
	return index
}

func main() {
	s := server.NewMCPServer(
		"Meilisearch Hybrid Search MCP Server",
		"1.0.0",
		server.WithResourceCapabilities(true, true),
		server.WithLogging(),
	)

	index := newMeiliIndex()

	settings, err := index.GetSettings()
	if err != nil {
		fmt.Printf("Error getting index settings: %v\n", err)
		os.Exit(1)
	}

	filterableAttrs := settings.FilterableAttributes

	filterableAttrDescription := "Attribute to filter on. Requires filter_word."
	if len(filterableAttrs) > 0 {
		availableAttrsStr := strings.Join(filterableAttrs, ", ")
		filterableAttrDescription = fmt.Sprintf("Attribute to filter on (Available: %s). Requires filter_word.", availableAttrsStr)
	}

	searchTool := mcp.NewTool("hybrid_search",
		mcp.WithDescription("Hybrid search your documents in Meilisearch index"),
		mcp.WithString("keywords",
			mcp.Required(),
			mcp.Description("Placing the most contextually important keywords at the beginning leads to more relevant results. (Good example: 'v1.13 new features meilisearch', Bad example: 'new features of meilisearch v1.13')"),
		),
		mcp.WithNumber("semantic_ratio",
			mcp.Required(),
			mcp.Description("A value closer to 0 emphasizes keyword search, while closer to 1 emphasizes vector search. Default is 0.5. If the `_rankingScore` in results is low, try adjusting to 0.8 or 0.2 to find more relevant documents"),
			mcp.DefaultNumber(0.5),
			mcp.Min(0.0),
			mcp.Max(1.0),
		),
		mcp.WithString("filterable_attribute",
			mcp.Description(filterableAttrDescription),
		),
		mcp.WithString("filter_word",
			mcp.Description("Word or value to filter the attribute by (e.g., 'Drama', 'Tolkien'). Requires filterable_attribute."),
		),
		mcp.WithNumber("ranking_score_threshold",
			mcp.Description("Returns results with a ranking score bigger than this value. Default is 0.9."),
			mcp.DefaultNumber(0.9),
			mcp.Min(0.0),
			mcp.Max(0.99),
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
		rankingScoreThresholdValue, ok := request.Params.Arguments["ranking_score_threshold"]
		if !ok {
			rankingScoreThresholdValue = 0.5
		}
		rankingScoreThreshold, ok := rankingScoreThresholdValue.(float64)
		if !ok {
			return nil, errors.New("argument 'ranking_score_threshold' must be a number")
		}

		var filterAttribute, filterWord string
		var filterExpr string

		filterAttrValue, filterAttrOk := request.Params.Arguments["filterable_attribute"]
		filterWordValue, filterWordOk := request.Params.Arguments["filter_word"]

		if filterAttrOk && filterWordOk {
			var attrOk, wordOk bool
			filterAttribute, attrOk = filterAttrValue.(string)
			filterWord, wordOk = filterWordValue.(string)

			if attrOk && wordOk && filterAttribute != "" && filterWord != "" {
				filterExpr = fmt.Sprintf(`%s = '%s'`, filterAttribute, filterWord)
			}
		}
		meiliEmbedder := *embedderFlag
		if meiliEmbedder == "" {
			meiliEmbedder = os.Getenv("MEILI_EMBEDDER")
			if meiliEmbedder == "" {
				return nil, errors.New("embedder not provided. Use --embedder flag or set MEILI_EMBEDDER environment variable")
			}
		}

		searchReq := &meilisearch.SearchRequest{
			Query: keywords,
			Hybrid: &meilisearch.SearchRequestHybrid{
				SemanticRatio: semanticRatio,
				Embedder:      meiliEmbedder,
			},
			Filter:                filterExpr,
			ShowRankingScore:      true,
			RankingScoreThreshold: rankingScoreThreshold,
		}

		searchRes, err := index.Search(keywords, searchReq)
		if err != nil {
			return nil, fmt.Errorf("meilisearch search failed: %w", err)
		}

		if len(searchRes.Hits) == 0 {
			return mcp.NewToolResultText("no results found - please try with fewer or different keywords"), nil
		}

		jsonResult, err := json.Marshal(searchRes.Hits)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal search result to JSON: %w", err)
		}

		return mcp.NewToolResultText(string(jsonResult)), nil
	})

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
