package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

var shellBuiltins = []string{"exit", "echo", "type", "pwd", "cd", "cat"}
var shellCommands = map[string]func([]string) (string, bool, error){
	"exit": runExit,
	"echo": runEcho,
	"type": runType,
	"pwd":  runPwd,
	"cd":   runCd,
	"cat":  runCat,
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
		processShellInput(input[:len(input)-1])
	}
}

func processShellInput(input string) {
	command, args, descriptor := defineCommandAndArgs(input)

	var output string
	var hasOutput bool = false
	var err error

	run, ok := shellCommands[command]

	if !ok {
		output, hasOutput, err = runExternal(command, args)
	} else {
		output, hasOutput, err = run(args)
	}

	if err != nil {
		fmt.Println(err)

		return
	}

	if !hasOutput {
		return
	}

	if descriptor.StdoutPath != "" {
		processOutputWithDescriptor(output, descriptor)

		return
	}
	fmt.Println(strings.TrimSuffix(output, "\n"))
}

// Shell builtins

func runExit(args []string) (string, bool, error) {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		os.Exit(1)
	}

	os.Exit(num)

	return "", false, nil
}

func runEcho(args []string) (string, bool, error) {
	return strings.Join(args, " "), true, nil
}

func runType(args []string) (string, bool, error) {
	command := args[0]
	// @TODO: Refactor this to use a map
	ok := slices.Contains(shellBuiltins, command)

	if ok {
		return command + " is a shell builtin", true, nil
	}

	externalCommand, ok := findExternal(command)
	if !ok {
		return "", false, errors.New(command + ": not found")
	}

	return command + " is " + externalCommand, true, nil
}

func runPwd(args []string) (string, bool, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("current directory could not be found")
	}

	return dir, true, nil
}

func runCd(args []string) (string, bool, error) {
	if len(args) < 1 {
		return "", false, fmt.Errorf("cd: missing operand")
	}

	path := args[0]

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", false, errors.New("home directory could not be found")
		}
		path = homeDir
	}

	err := os.Chdir(path)
	if err != nil {
		return "", false, fmt.Errorf("cd: %s: No such file or directory", path)
	}

	return "", true, nil
}

func runCat(args []string) (string, bool, error) {
	if len(args) < 1 {
		return "", false, errors.New("cat: missing operand")
	}

	var output string

	for _, filePath := range args {
		catOutput, ok, err := catFile(filePath)
		if !ok {
			return "", false, err
		}

		output += catOutput
	}

	return output, true, nil
}

// External commands

func runExternal(command string, input []string) (string, bool, error) {
	_, ok := findExternal(command)
	if !ok {
		return "", false, errors.New(command + ": command not found")
	}

	cmd := exec.Command(command, input...)

	output, err := cmd.Output()
	if err != nil {
		return "", false, errors.New("Error running external command:" + err.Error() + "\n")
	}

	return string(output), true, nil
}

// Utility functions

func catFile(path string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Println(logAllFilesInDir(filepath.Base(path)))

		return "", false, errors.New("cat: " + path + ": No such file or directory")
	}

	defer file.Close()

	scanner := bufio.NewScanner(file)
	var output string

	for scanner.Scan() {
		output += scanner.Text() + "\n"
	}

	return output, true, nil
}

func logAllFilesInDir(path string) string {
	files, err := os.ReadDir(path)
	if err != nil {
		return ""
	}

	var output string

	for _, file := range files {
		output += file.Name() + "\n"
	}

	return output
}

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

func defineCommandAndArgs(userInput string) (string, []string, Descriptor) {
	parsedInput := parseArguments(userInput)
	command := parsedInput[0]
	args := parsedInput[1:]

	descriptorIndex, descriptor := findDescriptor(args)

	return command, args[:descriptorIndex+1], descriptor
}

type Descriptor struct {
	StdoutPath string
}

func findDescriptor(args []string) (int, Descriptor) {
	var descriptor Descriptor = Descriptor{StdoutPath: ""}
	var argsLen int = len(args) - 1
	var descriptorIndex int = argsLen

	for i := argsLen; i >= 0; i-- {
		if args[i] == ">" || args[i] == "1>" {
			// Check if there is a path after the redirection operator
			if (i + 1) > argsLen {
				continue
			}
			descriptor.StdoutPath = args[i+1]
			descriptorIndex = i - 1
		}
	}

	return descriptorIndex, descriptor
}

func processOutputWithDescriptor(output string, descriptor Descriptor) bool {
	file, err := os.Create(descriptor.StdoutPath)
	if err != nil {
		return false
	}

	output = strings.TrimSuffix(output, "\n")

	file.WriteString(output)
	file.Close()

	return true
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
					currentArg += char
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
