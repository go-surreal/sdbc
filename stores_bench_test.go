package sdbc

import "testing"

func BenchmarkNewRequestKey0(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey0()
	}
}

func BenchmarkNewRequestKey1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey1()
	}
}

func BenchmarkNewRequestKey2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey2()
	}
}

func BenchmarkNewRequestKey4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey4()
	}
}

func BenchmarkNewRequestKey5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey5()
	}
}
