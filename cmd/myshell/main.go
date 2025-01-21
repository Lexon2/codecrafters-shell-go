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

var shellBuiltins = []string{"exit", "echo", "type", "pwd", "cd"}
var shellCommands = map[string]func([]string){
	"exit": runExit,
	"echo": runEcho,
	"type": runType,
	"pwd":  runPwd,
	"cd":   runCd,
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
		command, args := defineCommandAndArgs(input[:len(input)-1])

		run, ok := shellCommands[command]

		if !ok {
			runExternal(command, args)
		} else {
			run(args)
		}
	}
}

// Shell builtins

func runExit(args []string) {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		os.Exit(1)
	}

	os.Exit(num)
}

func runEcho(args []string) {
	fmt.Println(strings.Join(args, " "))
}

func runType(args []string) {
	command := args[0]
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

func runPwd(args []string) {
	dir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current directory")
		return
	}

	fmt.Println(dir)
}

func runCd(args []string) {
	if len(args) < 1 {
		fmt.Println("cd: missing operand")
		return
	}

	path := args[0]

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Println("Error getting home directory")
			return
		}
		path = homeDir
	}

	err := os.Chdir(path)
	if err != nil {
		fmt.Println("cd: " + path + ": No such file or directory")
	}
}

// 			fmt.Println("Error reading file")
// 		}
// 	}
// }

// External commands

func runExternal(command string, input []string) {
	_, ok := findExternal(command)
	if !ok {
		fmt.Println(command + ": command not found")
		return
	}

	if command == "cat" {
		// filesInFolder := findFilesInFolder(input[0])
		// fileNamesMatches := []string{}
		// for i, filePath := range input {
		// 	fileName := filepath.Base(filePath)

		// 	fileNamesMatches = append(fileNamesMatches, fileName+", "+filesInFolder[i])
		// }

		// fmt.Println(strings.Join(fileNamesMatches, " | "))

		var catOutput string = ""
		for _, file := range input {
			cmd := exec.Command(command, file)
			output, err := cmd.Output()
			if err != nil {
				fmt.Printf("Error on executing external command: %s\n", err)
				return
			}

			catOutput += string(output)
		}
		fmt.Print(catOutput)

		return
	}
	cmd := exec.Command(command, input...)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error on executing external command: %s\n", err)
		return
	}

	fmt.Print(string(output))
}

// Utility functions

// func findFilesInFolder(filePath string) []string {
// 	dirPath := filepath.Dir(filePath)
// 	files, err := os.ReadDir(dirPath)
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 	}

// 	var result []string = []string{}
// 	for _, file := range files {
// 		result = append(result, file.Name())
// 	}

// 	return result
// }

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

func defineCommandAndArgs(userInput string) (string, []string) {
	splitted := strings.Split(userInput, " ")
	command := splitted[0]

	// fmt.Println(strings.Join(parseArguments(strings.Join(splitted[1:], " ")), ","))

	return command, parseArguments(strings.Join(splitted[1:], " "))
}

func parseArguments(argsInput string) []string {
	if len(argsInput) == 0 {
		return []string{}
	}

	var stack []string = strings.Split(argsInput, "")
	slices.Reverse(stack)
	var stackLen int = len(stack)

	var result []string
	var currentArg string = ""

	var isSingleQuoteArg bool = false
	var isDoubleQuoteArg bool = false
	var hasSpace bool = false

	for stackLen > 0 {
		char := stack[stackLen-1]
		stackLen--

		switch char {
		case " ":
			if isSingleQuoteArg || isDoubleQuoteArg {
				currentArg += char
				continue
			}

			if hasSpace {
				continue
			}

			if currentArg == "" {
				continue
			}

			hasSpace = true
			result = append(result, currentArg)
			currentArg = ""

		case "'":
			if isDoubleQuoteArg {
				currentArg += char
				continue
			}

			if isSingleQuoteArg {
				if stackLen == 0 || stack[stackLen-1] == " " {
					isSingleQuoteArg = false
					continue
				}
			}

			isSingleQuoteArg = !isSingleQuoteArg

		case "\"":
			if isSingleQuoteArg {
				currentArg += char
				continue
			}

			if isDoubleQuoteArg {
				if stackLen == 0 || stack[stackLen-1] != " " {
					isDoubleQuoteArg = false
					continue
				}
			}

			isDoubleQuoteArg = !isDoubleQuoteArg

		case "\\":
			if stackLen == 0 {
				continue
			}

			if isSingleQuoteArg {
				currentArg += char
				continue
			}

			nextChar := stack[stackLen-1]

			if isDoubleQuoteArg {
				if nextChar != "\"" && nextChar != "\\" && nextChar != "$" {
					continue
				}
			}

			currentArg += nextChar
			stackLen--
		default:
			currentArg += char
		}

		if hasSpace {
			hasSpace = false
		}
	}

	if currentArg != "" {
		result = append(result, currentArg)
	}

	return result
}
