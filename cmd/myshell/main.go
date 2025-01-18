package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// Ensures gofmt doesn't remove the "fmt" import in stage 1 (feel free to remove this!)
var _ = fmt.Fprint

var shellBuiltins = []string{"exit", "echo", "type", "pwd"}
var shellCommands = map[string]func([]string){
	"exit": runExit,
	"echo": runEcho,
	"type": runType,
	"pwd":  runPwd,
}

func main() {
	for {
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
			runExternal(splitted)
		} else {
			run(splitted)
		}
	}
}

// Shell builtins

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

	externalCommand, ok := findExternal(command)
	if !ok {
		fmt.Println(command + ": not found")
		return
	}

	fmt.Println(command + " is " + externalCommand)
}

func runPwd(input []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory")
		return
	}

	fmt.Println(dir)
}

// External commands

func runExternal(input []string) {
	command := input[0]

	_, ok := findExternal(command)
	if !ok {
		fmt.Println(command + ": command not found")
		return
	}

	cmd := exec.Command(command, input[1:]...)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Print(string(output))
}

// Utility functions

func findExternal(command string) (string, bool) {
	paths := os.Getenv("PATH")
	separator := getEnvPathSeparator()

	for _, path := range strings.Split(paths, separator) {
		pathToCommand := filepath.Join(path, command)

		if _, err := os.Stat(pathToCommand); err == nil {
			return pathToCommand, true
		}
	}

	return "", false
}

func getEnvPathSeparator() string {
	os := runtime.GOOS
	switch os {
	case "windows":
		return ";"
	default:
		return ":"
	}
}
