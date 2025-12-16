package search

import (
	"real-estate-portal/internal/models"

	"github.com/meilisearch/meilisearch-go"
)

type SearchClient struct {
	client *meilisearch.Client
	index  string
}

func NewSearchClient(host, apiKey string) *SearchClient {
	client := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   host,
		APIKey: apiKey,
	})

	return &SearchClient{
		client: client,
		index:  "properties",
	}
}

// InitIndex initializes the Meilisearch index
func (s *SearchClient) InitIndex() error {
	// Create index if it doesn't exist
	_, err := s.client.CreateIndex(&meilisearch.IndexConfig{
		Uid:        s.index,
		PrimaryKey: "id",
	})
	// Ignore error if index already exists
	if err != nil && err.Error() != "index already exists" {
		return err
	}

	// Configure searchable attributes
	_, err = s.client.Index(s.index).UpdateSearchableAttributes(&[]string{
		"title",
		"detail_url",
		"station",
		"address",
		"floor_plan",
	})
	if err != nil {
		return err
	}

	// Configure filterable attributes
	_, err = s.client.Index(s.index).UpdateFilterableAttributes(&[]string{
		"id",
		"rent",
		"floor_plan",
		"walk_time",
		"area",
		"building_age",
		"floor",
		"station",
	})
	if err != nil {
		return err
	}

	// Configure sortable attributes
	_, err = s.client.Index(s.index).UpdateSortableAttributes(&[]string{
		"rent",
		"area",
		"walk_time",
		"building_age",
		"created_at",
	})
	if err != nil {
		return err
	}

	return nil
}

// IndexProperty indexes a single property
func (s *SearchClient) IndexProperty(property *models.Property) error {
	_, err := s.client.Index(s.index).AddDocuments([]models.Property{*property})
	return err
}

// IndexProperties indexes multiple properties
func (s *SearchClient) IndexProperties(properties []models.Property) error {
	if len(properties) == 0 {
		return nil
	}
	_, err := s.client.Index(s.index).AddDocuments(properties)
	return err
}

// SearchRequest represents advanced search parameters
type SearchRequest struct {
	Query           string
	Limit           int64
	Offset          int64
	Filter          []string
	Sort            []string
	FacetsFilter    []string
	AttributesToRetrieve []string
}

// SearchResult represents search results with facets
type SearchResult struct {
	Hits       []models.Property
	TotalHits  int64
	Facets     map[string]interface{}
	ProcessingTime int64
}

// Search searches for properties with basic options
func (s *SearchClient) Search(query string, limit int64) ([]models.Property, error) {
	result, err := s.AdvancedSearch(SearchRequest{
		Query: query,
		Limit: limit,
	})
	if err != nil {
		return nil, err
	}
	return result.Hits, nil
}

// AdvancedSearch performs advanced search with facets and filters
func (s *SearchClient) AdvancedSearch(req SearchRequest) (*SearchResult, error) {
	if req.Limit == 0 {
		req.Limit = 20
	}

	searchReq := &meilisearch.SearchRequest{
		Limit:  req.Limit,
		Offset: req.Offset,
	}

	// Add filters
	if len(req.Filter) > 0 {
		filterStr := ""
		for i, f := range req.Filter {
			if i > 0 {
				filterStr += " AND "
			}
			filterStr += f
		}
		searchReq.Filter = filterStr
	}

	// Add sorting
	if len(req.Sort) > 0 {
		searchReq.Sort = req.Sort
	}

	// Add facets
	if len(req.FacetsFilter) > 0 {
		searchReq.Facets = req.FacetsFilter
	}

	// Add attributes to retrieve
	if len(req.AttributesToRetrieve) > 0 {
		searchReq.AttributesToRetrieve = req.AttributesToRetrieve
	}

	searchRes, err := s.client.Index(s.index).Search(req.Query, searchReq)
	if err != nil {
		return nil, err
	}

	properties := make([]models.Property, 0, len(searchRes.Hits))
	for _, hit := range searchRes.Hits {
		property := parsePropertyFromHit(hit)
		properties = append(properties, property)
	}

	var facets map[string]interface{}
	if searchRes.FacetDistribution != nil {
		facets, _ = searchRes.FacetDistribution.(map[string]interface{})
	}

	result := &SearchResult{
		Hits:           properties,
		TotalHits:      searchRes.EstimatedTotalHits,
		Facets:         facets,
		ProcessingTime: searchRes.ProcessingTimeMs,
	}

	return result, nil
}

// parsePropertyFromHit converts a search hit to a Property
func parsePropertyFromHit(hit interface{}) models.Property {
	hitMap := hit.(map[string]interface{})
	property := models.Property{
		ID:        getString(hitMap, "id"),
		DetailURL: getString(hitMap, "detail_url"),
		Title:     getString(hitMap, "title"),
		ImageURL:  getString(hitMap, "image_url"),
		Station:   getString(hitMap, "station"),
		Address:   getString(hitMap, "address"),
		FloorPlan: getString(hitMap, "floor_plan"),
		Status:    models.PropertyStatus(getString(hitMap, "status")),
	}

	// Parse numeric fields
	if rent, ok := hitMap["rent"].(float64); ok {
		rentInt := int(rent)
		property.Rent = &rentInt
	}
	if area, ok := hitMap["area"].(float64); ok {
		property.Area = &area
	}
	if walkTime, ok := hitMap["walk_time"].(float64); ok {
		walkTimeInt := int(walkTime)
		property.WalkTime = &walkTimeInt
	}
	if buildingAge, ok := hitMap["building_age"].(float64); ok {
		buildingAgeInt := int(buildingAge)
		property.BuildingAge = &buildingAgeInt
	}
	if floor, ok := hitMap["floor"].(float64); ok {
		floorInt := int(floor)
		property.Floor = &floorInt
	}

	return property
}

// getString safely extracts a string from map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// GetFacets retrieves facet distribution for specified fields
func (s *SearchClient) GetFacets(facets []string) (map[string]interface{}, error) {
	searchRes, err := s.client.Index(s.index).Search("", &meilisearch.SearchRequest{
		Limit:  0,
		Facets: facets,
	})
	if err != nil {
		return nil, err
	}

	if searchRes.FacetDistribution != nil {
		if facetMap, ok := searchRes.FacetDistribution.(map[string]interface{}); ok {
			return facetMap, nil
		}
	}
	return map[string]interface{}{}, nil
}
