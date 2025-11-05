package ncservice

// strPtr is helper to get pointer to string
// Example: strPtr("hello")
func Ptr[T any](v T) *T {
	return &v
}
