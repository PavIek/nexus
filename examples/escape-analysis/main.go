package main

func allocate() *int {
	x := 42
	return &x
}

func noEscape() int {
	x := 42
	return x
}

func closureEscape() func() int {
	x := 5
	return func() int { return x } // x escapes
}

var global *int

func assignGlobal() {
	x := 7
	global = &x // escapes
}

func makeLargeSlice() []int {
	s := make([]int, 10000) // may escape due to size
	return s
}

func main() {
	a := allocate()
	b := noEscape()
	c := closureEscape()
	assignGlobal()
	d := makeLargeSlice()

	println(*a)
	println(b)
	println(c())
	println(*global)
	println(len(d))
}
