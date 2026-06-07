package catalog

// Store is the catalog container owned by a single account.
type Store struct {
	ID      string
	OwnerID string
	Name    string
}

// Category groups Products within a Store's catalog.
type Category struct {
	ID      string
	StoreID string
	Name    string
}

// Product is a sellable item placed under a Category.
type Product struct {
	ID         string
	StoreID    string
	CategoryID string
	Name       string
	Price      float64
}

// Page is a single page of a list endpoint's results, alongside enough
// metadata for a client to render pagination controls without a second request.
type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// newPage slices items into the requested page and fills in the metadata.
// total is the count of all matching items (before slicing), so TotalPages
// reflects the full result set rather than just what's returned.
func newPage[T any](items []T, page, limit, total int) Page[T] {
	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}

	skip := (page - 1) * limit
	if skip > total {
		skip = total
	}

	end := skip + limit
	if end > total {
		end = total
	}

	return Page[T]{
		Items:      items[skip:end],
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}
}
