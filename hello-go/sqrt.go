package main

import "fmt"

func sqrt(x float64) float64 {
	z := 1.0
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}

func main() {
	for i := 0; i < 10; i++ {
		fmt.Println(sqrt(float64(i)))
	}
}
