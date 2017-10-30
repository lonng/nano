package nano

// Invoke invokes function in main logic goroutine
func Invoke(fn func()) {
	handler.chFunction <- fn
}
