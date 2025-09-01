package storecmn

type CommonFindParams struct {
	IsAscOrdering bool   `json:"is_asc_ordering"`
	OrderBy       string `json:"order_by"`
	PageParams
}

func NewCommonFindParams() *CommonFindParams {
	return &CommonFindParams{}
}

func (s *CommonFindParams) SetIsAscOrdering(v bool) *CommonFindParams {
	s.IsAscOrdering = v
	return s
}

func (s *CommonFindParams) SetOrderBy(v string) *CommonFindParams {
	s.OrderBy = v
	return s
}

func (s *CommonFindParams) SetPage(v *uint64) *CommonFindParams {
	s.Page = v
	return s
}

func (s *CommonFindParams) SetPageSize(v *uint64) *CommonFindParams {
	s.PageSize = v
	return s
}

type PageParams struct {
	Page     *uint64
	PageSize *uint64
}

type FindResponse[T any] struct {
	Items []T `json:"items"`
}

type FindResponseWithPagingFlag[T any] struct {
	Items            []T  `json:"items"`
	IsNextPageExists bool `json:"is_next_page_exists"`
}
