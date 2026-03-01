package bitio

func mask64(n int) uint64 {
	if n == 64 {
		return ^uint64(0)
	}
	return (1 << n) - 1
}
