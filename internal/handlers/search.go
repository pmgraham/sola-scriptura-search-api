package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/sola-scriptura-search-api/internal/models"
	"github.com/sola-scriptura-search-api/internal/services"
)

// SearchHandler handles search endpoints
type SearchHandler struct {
	vectorSearch *services.VectorSearchService
}

// NewSearchHandler creates a new search handler
func NewSearchHandler(vectorSearch *services.VectorSearchService) *SearchHandler {
	return &SearchHandler{
		vectorSearch: vectorSearch,
	}
}

// SemanticSearch handles POST /search - semantic verse search
func (h *SearchHandler) SemanticSearch(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.SemanticSearchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Query is required")
	}

	limit := req.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	citations, err := h.vectorSearch.SearchVersesCitations(ctx, req.Query, limit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Search failed: "+err.Error())
	}

	return c.JSON(http.StatusOK, models.SemanticSearchResponse{
		Query:   req.Query,
		Results: citations,
	})
}

// HybridSearch handles POST /search/hybrid - searches both verses and topics
func (h *SearchHandler) HybridSearch(c echo.Context) error {
	ctx := c.Request().Context()

	var req models.HybridSearchRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Query is required")
	}

	verseLimit := req.VerseLimit
	if verseLimit <= 0 || verseLimit > 50 {
		verseLimit = 10
	}

	topicLimit := req.TopicLimit
	if topicLimit <= 0 || topicLimit > 50 {
		topicLimit = 5
	}

	// Search verses
	citations, err := h.vectorSearch.SearchVersesCitations(ctx, req.Query, verseLimit)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Search failed: "+err.Error())
	}

	// Search topics by keywords
	topics, err := h.vectorSearch.SearchTopics(ctx, req.Query, topicLimit)
	if err != nil {
		c.Logger().Warnf("Topic search failed: %v", err)
		topics = []models.ScoredTopic{}
	}

	return c.JSON(http.StatusOK, models.HybridSearchResponse{
		Query: req.Query,
		ResourceMatches: models.ResourceMatches{
			Topics: topics,
		},
		SemanticMatches: models.SemanticMatches{
			Verses: citations,
		},
	})
}

// RegisterRoutes registers search routes
func (h *SearchHandler) RegisterRoutes(g *echo.Group) {
	g.POST("/search", h.SemanticSearch)
	g.POST("/search/hybrid", h.HybridSearch)
}
