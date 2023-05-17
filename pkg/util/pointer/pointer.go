package pointer

func New[T any](t T) *T {
	return &t
}
