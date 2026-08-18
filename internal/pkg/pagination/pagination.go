package pagination

// 列表分页约定：默认第 1 页、每页 10 条；前端表格可选 10/20/50/100。
// API 仍允许 1～MaxPageSize（例如用 page_size=1 只取 total）。
const (
	DefaultPage     = 1
	DefaultPageSize = 10
	MaxPageSize     = 100
)

type Result[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

func Normalize(page, pageSize int) (int, int) {
	if page <= 0 {
		page = DefaultPage
	}
	if pageSize <= 0 {
		pageSize = DefaultPageSize
	}
	if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}
	return page, pageSize
}
