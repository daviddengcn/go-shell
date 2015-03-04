package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/daviddengcn/go-shell/vm"
)

var (
	PS = "$ "
)

func main() {
	fmt.Println("go-shell 1.0")
	vm := gsvm.New()

	in := bufio.NewReader(os.Stdin)

	buffered := ""

	for {
		if buffered == "" {
			fmt.Print(PS)
		}
		line, err := in.ReadString('\n')
		if err != nil {
			fmt.Println()
			if err == io.EOF {
				return
			}
			log.Fatalf("Read error: %v", err)
		}
		isFragment := vm.Run(buffered + line)
		if isFragment {
			buffered += "\n" + line
		} else {
			buffered = ""
		}
	}
}
