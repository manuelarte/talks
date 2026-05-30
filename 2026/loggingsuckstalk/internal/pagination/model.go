package pagination

type (
	PageRequest struct {
		Number int
		Size   int
	}

	Page[T any] struct {
		Content    []T
		TotalCount int64
		TotalPages int
		Number     int
		Size       int
	}
)
