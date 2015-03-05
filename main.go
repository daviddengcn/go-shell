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
		err = vm.Run(buffered + line)
		if err == gsvm.FragmentErr {
			buffered += "\n" + line
		} else {
			if err != nil {
				log.Println(err)
			}
			buffered = ""
		}
	}
}
