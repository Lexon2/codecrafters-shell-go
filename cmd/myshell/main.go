package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

var shellBuiltins = []string{"exit", "echo", "type"}
var shellCommands = map[string]func([]string){
	"exit": runExit,
	"echo": runEcho,
	"type": runType,
}

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

		run, ok := shellCommands[command]

		if !ok {
			fmt.Println(command + ": command not found")
			os.Exit(1)
		}

		run(splitted)
	}
}

func runExit(input []string) {
	num, err := strconv.Atoi(input[1])
	if err != nil {
		os.Exit(1)
	}

	os.Exit(num)
}

func runEcho(input []string) {
	fmt.Println(strings.Join(input[1:], " "))
}

func runType(input []string) {
	command := input[1]
	ok := slices.Contains(shellBuiltins, command)

	if ok {
		fmt.Println(command + " is a shell builtin")
		return
	}

	paths := os.Getenv("PATH")

	for _, path := range strings.Split(paths, ":") {
		pathToCommand := filepath.Join(path, command)

		if _, err := os.Stat(pathToCommand); err == nil {
			fmt.Println(command + " is " + pathToCommand)
			return
		}
	}

	fmt.Println(command + ": not found")
}
