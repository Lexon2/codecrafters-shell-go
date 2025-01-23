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

var shellBuiltins = []string{"exit", "echo", "type", "pwd", "cd"}
var shellCommands = map[string]func([]string) []CommandResult{
	"exit": runExit,
	"echo": runEcho,
	"type": runType,
	"pwd":  runPwd,
	"cd":   runCd,
	"cat":  runCat,
}

type CommandResult struct {
	Output    string
	HasOutput bool
	Err       error
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
	run, ok := shellCommands[command]

	var commandResults []CommandResult = nil

	if !ok {
		commandResults = runExternal(command, args)
	} else {
		commandResults = run(args)
	}

	var outputs []string = nil
	for _, result := range commandResults {
		if result.Err != nil {
			fmt.Println(result.Err)

			continue
		}

		if !result.HasOutput {
			continue
		}

		outputs = append(outputs, strings.TrimSuffix(result.Output, "\n"))
	}

	if len(outputs) == 0 {
		return
	}

	output := strings.Join(outputs, "")

	if descriptor.StdoutPath != "" {
		processOutputWithDescriptor(output, descriptor)

		return
	}
	fmt.Println(strings.TrimSuffix(output, "\n"))

}

// Shell builtins

func runExit(args []string) []CommandResult {
	num, err := strconv.Atoi(args[0])
	if err != nil {
		os.Exit(1)
	}

	os.Exit(num)

	return []CommandResult{{Output: "", HasOutput: false, Err: nil}}
}

func runEcho(args []string) []CommandResult {
	return []CommandResult{{Output: strings.Join(args, " "), HasOutput: true, Err: nil}}
}

func runType(args []string) []CommandResult {
	command := args[0]
	// @TODO: Refactor this to use a map
	ok := slices.Contains(shellBuiltins, command)

	if ok {
		return []CommandResult{{Output: command + " is a shell builtin", HasOutput: true, Err: nil}}
	}

	externalCommand, ok := findExternal(command)
	if !ok {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New(command + ": not found")}}
	}

	return []CommandResult{{Output: command + " is " + externalCommand, HasOutput: true, Err: nil}}
}

func runPwd(args []string) []CommandResult {
	dir, err := os.Getwd()
	if err != nil {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New("current directory could not be found")}}
	}

	return []CommandResult{{Output: dir, HasOutput: true, Err: nil}}
}

func runCd(args []string) []CommandResult {
	if len(args) < 1 {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New("cd: missing operand")}}
	}

	path := args[0]

	if path == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return []CommandResult{{Output: "", HasOutput: false, Err: errors.New("home directory could not be found")}}
		}
		path = homeDir
	}

	err := os.Chdir(path)
	if err != nil {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New("cd: " + path + ": No such file or directory")}}
	}

	return []CommandResult{{Output: "", HasOutput: false, Err: nil}}
}

func runCat(args []string) []CommandResult {
	if len(args) < 1 {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New("cat: missing operand")}}
	}

	var result []CommandResult = nil

	for _, filePath := range args {
		catOutput, ok, err := catFile(filePath)
		if !ok {
			result = append(result, CommandResult{Output: "", HasOutput: false, Err: err})
			continue
		}

		result = append(result, CommandResult{Output: catOutput, HasOutput: true, Err: nil})
	}

	return result
}

// External commands

func runExternal(command string, args []string) []CommandResult {
	_, ok := findExternal(command)
	if !ok {
		return []CommandResult{{Output: "", HasOutput: false, Err: errors.New(command + ": command not found")}}
	}

	result := []CommandResult{}

	for _, arg := range args {
		cmd := exec.Command(command, arg)
		output, err := cmd.Output()

		if err != nil {
			switch command {
			case "cat":
				result = append(result, CommandResult{Output: "", HasOutput: false, Err: errors.New("cat: " + arg + ": No such file or directory")})
			default:
				result = append(result, CommandResult{Output: "", HasOutput: false, Err: errors.New("Error running external command:" + err.Error() + "\n")})
			}
		}

		result = append(result, CommandResult{Output: string(output), HasOutput: true, Err: nil})
	}

	return result
}

// Utility functions

func catFile(path string) (string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
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

// func logAllFilesInDir(path string) string {
// 	files, err := os.ReadDir(path)
// 	if err != nil {
// 		return ""
// 	}

// 	var output string

// 	for _, file := range files {
// 		output += file.Name() + "\n"
// 	}

// 	return output
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
