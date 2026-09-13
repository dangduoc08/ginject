package catalog

type Store struct {
	ID      string
	OwnerID string
	Name    string
}

type Category struct {
	ID      string
	StoreID string
	Name    string
}

type Product struct {
	ID         string
	StoreID    string
	CategoryID string
	Name       string
	Price      float64
}

type Page[T any] struct {
	Items      []T `json:"items"`
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

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
