package list

import (
	"github.com/dangduoc08/gooh"
	"github.com/dangduoc08/gooh/common"
)

type ReadCompaniesQuery struct {
	Name        string      `bind:"name"`
	SalaryRange []complex64 `bind:"salary_range"`
	IsPublished bool        `bind:"is_published"`
	Limit       int         `bind:"limit"`
	Offset      int         `bind:"offset"`
	Order       string      `bind:"order"`
	Sort        string      `bind:"sort"`
}

func (dto ReadCompaniesQuery) Transform(query gooh.Query, metadata common.ArgumentMetadata) any {

	return query.Bind(dto)
}

type CreateListBodySub struct {
	IsMale bool `bind:"isMale"`
}

type CreateListBody struct {
	Title      string            `bind:"title"`
	TitleColor string            `bind:"title_color"`
	Sub        CreateListBodySub `bind:"subNe"`
	// Limit      int    `bind:"limit"`
	// Offset     int    `bind:"offset"`
	// Order      string `bind:"order"`
	// Sort       string `bind:"sort"`
}

func (dto CreateListBody) Transform(form gooh.Body, metadata common.ArgumentMetadata) any {
	return form.Bind(dto)
}

type CreateListQuery struct {
	Limit  int `bind:"limit"`
	Offset int `bind:"offset"`
}

func (dto CreateListQuery) Transform(query gooh.Query, metadata common.ArgumentMetadata) any {
	return query.Bind(dto)
}
