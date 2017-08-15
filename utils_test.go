package nano

import "testing"

func null(key, value int) int {
	key += value
	return key
}

func test1(m map[int]int) {
	if len(m) < 1 {
		return
	}
	for k, v := range m {
		null(k, v)
	}
}

func BenchmarkEmptyMap1(b *testing.B) {
	b.ReportAllocs()

	m := map[int]int{}
	for i := 0; i < b.N; i++ {
		test1(m)
	}
}

func test2(m map[int]int) {
	for k, v := range m {
		null(k, v)
	}
}

func BenchmarkEmptyMap2(b *testing.B) {
	b.ReportAllocs()

	m := map[int]int{}
	for i := 0; i < b.N; i++ {
		test2(m)
	}
}
