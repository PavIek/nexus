package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	f, _ := os.Create("output.txt")
	defer f.Close()

	for i := range 10_000 {
		f.Write([]byte("line\n"))
		fmt.Println(i)
	}

	f, _ = os.Create("output.txt")
	defer f.Close()

	buf := bufio.NewWriter(f)
	for i := range 10_000 {
		buf.WriteString("line\n")
		fmt.Println(i)
	}
	buf.Flush() // ensure all buffered data is written

	f, _ = os.Create("output.txt")
	buf = bufio.NewWriterSize(f, 16*1024) // 16 KB buffer

	reader := bufio.NewReaderSize(f, 32*1024)
	fmt.Println(reader)
}
