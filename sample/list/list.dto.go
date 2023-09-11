package list

import (
	"fmt"

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

type CreateListBody struct {
	Title      string `bind:"title"`
	TitleColor string `bind:"title_color"`
	// Limit      int    `bind:"limit"`
	// Offset     int    `bind:"offset"`
	// Order      string `bind:"order"`
	// Sort       string `bind:"sort"`
}

func (dto CreateListBody) Transform(form gooh.Form, metadata common.ArgumentMetadata) any {
	fmt.Println("transform data type", metadata.ParamType)
	return form.Bind(dto)
}
