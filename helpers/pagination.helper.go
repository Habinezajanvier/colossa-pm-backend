package helpers

// PaginationParams holds query params for paginated requests
type PaginationParams struct {
	Page  int `form:"page"`
	Limit int `form:"limit"`
}

func (p *PaginationParams) Normalize() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 || p.Limit > 100 {
		p.Limit = 20
	}
}

func (p *PaginationParams) Offset() int {
	return (p.Page - 1) * p.Limit
}

// PaginatedResult is a generic paginated response wrapper
type PaginatedResult[T any] struct {
	Data        []T   `json:"data"`
	Total       int64 `json:"total"`
	Page        int   `json:"page"`
	Limit       int   `json:"limit"`
	TotalPages  int   `json:"totalPages"`
	HasNext     bool  `json:"hasNext"`
	HasPrevious bool  `json:"hasPrevious"`
}

// NewPaginatedResult builds a PaginatedResult from a slice, total count and params
func NewPaginatedResult[T any](data []T, total int64, params PaginationParams) *PaginatedResult[T] {
	totalPages := int(total) / params.Limit
	if int(total)%params.Limit != 0 {
		totalPages++
	}

	return &PaginatedResult[T]{
		Data:        data,
		Total:       total,
		Page:        params.Page,
		Limit:       params.Limit,
		TotalPages:  totalPages,
		HasNext:     params.Page < totalPages,
		HasPrevious: params.Page > 1,
	}
}
