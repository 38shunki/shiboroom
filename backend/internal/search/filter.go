package search

import (
	"encoding/json"
	"fmt"
	"real-estate-portal/internal/models"
	"strings"

	"github.com/meilisearch/meilisearch-go"
)

type FilterParams struct {
	Query       string
	MinRent     *int
	MaxRent     *int
	FloorPlans  []string
	MaxWalkTime *int
	SortBy      string
	Limit       int64
}

// FilterSearch performs advanced search with filters
func (s *SearchClient) FilterSearch(params FilterParams) ([]models.Property, error) {
	var filters []string

	// Rent range filter
	if params.MinRent != nil {
		filters = append(filters, fmt.Sprintf("rent >= %d", *params.MinRent))
	}
	if params.MaxRent != nil {
		filters = append(filters, fmt.Sprintf("rent <= %d", *params.MaxRent))
	}

	// Floor plan filter
	if len(params.FloorPlans) > 0 {
		planFilters := make([]string, len(params.FloorPlans))
		for i, plan := range params.FloorPlans {
			planFilters[i] = fmt.Sprintf("floor_plan = '%s'", plan)
		}
		filters = append(filters, fmt.Sprintf("(%s)", strings.Join(planFilters, " OR ")))
	}

	// Walk time filter
	if params.MaxWalkTime != nil {
		filters = append(filters, fmt.Sprintf("walk_time <= %d", *params.MaxWalkTime))
	}

	// Combine filters
	var filterStr string
	if len(filters) > 0 {
		filterStr = strings.Join(filters, " AND ")
	}

	// Determine sort order
	var sort []string
	if params.SortBy != "" {
		sort = []string{params.SortBy}
	}

	// Default limit
	if params.Limit == 0 {
		params.Limit = 20
	}

	// Perform search
	searchReq := &meilisearch.SearchRequest{
		Limit: params.Limit,
	}

	if filterStr != "" {
		searchReq.Filter = filterStr
	}

	if len(sort) > 0 {
		searchReq.Sort = sort
	}

	searchRes, err := s.client.Index(s.index).Search(params.Query, searchReq)
	if err != nil {
		return nil, err
	}

	// Convert hits to properties
	var properties []models.Property
	for _, hit := range searchRes.Hits {
		// Convert hit to JSON then to Property struct
		hitJSON, err := json.Marshal(hit)
		if err != nil {
			continue
		}

		var property models.Property
		if err := json.Unmarshal(hitJSON, &property); err != nil {
			continue
		}

		properties = append(properties, property)
	}

	return properties, nil
}
