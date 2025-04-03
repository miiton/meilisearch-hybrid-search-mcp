# Meilisearch Hybrid Search MCP Server

This MCP (Model Control Protocol) server provides a tool for performing hybrid searches on a Meilisearch index. It allows combining keyword-based search with semantic vector search.

## Environment Variables

Set the following environment variables before running the server:

```bash
export MEILI_HOST="http://your-meilisearch-instance:7700" # Meilisearch host URL
export MEILI_API_KEY="your_api_key"                     # Meilisearch API key (if required)
export MEILI_INDEX="your_index_name"                    # The name of the index to search in
export MEILI_EMBEDDER="your_embedder_name"              # The name of the embedder configured in Meilisearch (e.g., 'default', 'myOpenai')
export MEILI_FILTERABLE_ATTRIBUTES="attr1,attr2"        # Comma-separated filterable attributes for AI awareness (from index settings)
```

## Building and Running

Build the server:
```bash
go build -o meilisearch-hybrid-search-mcp .

# windows
GOOS=windows GOARCH=amd64 go build -o meilisearch-hybrid-search-mcp.exe .
# linux
GOOS=linux GOARCH=amd64 go build -o meilisearch-hybrid-search-mcp .
# mac
GOOS=macos GOARCH=amd64 go build -o meilisearch-hybrid-search-mcp .
```

Run the server:
```bash
./meilisearch-hybrid-search-mcp
```
The server will listen on standard input/output.

## Available Tool: `hybrid_search`

This tool performs a hybrid search on the configured Meilisearch index.

**Description:** Hybrid search your documents in Meilisearch index.

**Arguments:**

*   `keywords` (string, **required**): The search query keywords.
*   `semantic_ratio` (number, optional, default: 0.5): Controls the balance between keyword and semantic search.
    *   `0.0`: Pure keyword search.
    *   `1.0`: Pure semantic search.
    *   `0.5`: Balanced keyword and semantic search.
*   `filterable_attribute` (string, optional): The attribute name to filter results on (e.g., "genre", "author"). Requires `filter_word`.
*   `filter_word` (string, optional): The value to filter the specified `filterable_attribute` by (e.g., "Drama", "Tolkien"). Requires `filterable_attribute`.

## Available Prompts

These prompts provide guidance on using the `hybrid_search` tool effectively.

*   **`adjust_semantic_ratio`**: Provides guidance on how to adjust the `semantic_ratio` parameter based on your search goal (keyword-focused, semantic-focused, or balanced).
    *   **Usage:** `prompts/get(name="adjust_semantic_ratio", arguments={"search_type": "semantic"})` (Replace "semantic" with "keyword" or omit for balanced guidance).
*   **`hybrid_search_help`**: Offers a general overview of the `hybrid_search` tool, including basic usage, semantic ratio adjustment, and filtering.
    *   **Usage:** `prompts/get(name="hybrid_search_help")`