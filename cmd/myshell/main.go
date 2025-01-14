package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

func main() {
	for {
		// Uncomment this block to pass the first stage
		fmt.Fprint(os.Stdout, "$ ")

		// Wait for user input
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')

		if err != nil {
			os.Exit(1)
		}

		// len(command)-1 removes the newline character
		splitted := strings.Split(input[:len(input)-1], " ")
		command := splitted[0]

		switch command {
		case "exit":
			runExit(splitted)
		default:
			fmt.Println(command + ": command not found")
		}
	}
}

func runExit(input []string) {
	num, err := strconv.Atoi(input[1])
	if err != nil {
		os.Exit(1)
	}

	os.Exit(num)
}
