package main

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kballard/go-shellquote"
	"github.com/spf13/cobra"
)

const (
	prompt = "[Yourname@Your PC]$ "
)

func setupStdin() {
	//no buffering
	err := exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()
	if err != nil {
		log.Fatalln("fatal error: cannot disable shell buffering")
	}
	//no visible output
	err = exec.Command("stty", "-F", "/dev/tty", "-echo").Run()
	if err != nil {
		log.Fatalln("fatal error: cannot disable auto echo")
	}

}

func readStdin(out chan string) {
	//str, err := exec.Command("ls", "-gah").CombinedOutput()
	//if err != nil {
	//	println(err.Error())
	//}
	//fmt.Println(string(str))

	setupStdin()

	b := make([]byte, 1)
	for {
		os.Stdin.Read(b)
		//println(b[0])
		out <- string(b)
	}
}

func resetStdin() {
	exec.Command("stty", "-F", "/dev/tty", "echo").Run()
}

func disableCtrlC() {
	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for {
			<-c
			println("Ctrl c pressed")
			os.Exit(1)
		}
	}()
}

func backspace(s string) string {
	size := len(s)
	if size == 0 {
		return s
	} else {
		return s[:size-1]
	}
}

func readCommand(stdin chan string) string {
	command := ""
charLoop:
	for {
		// blockin
		char := <-stdin

		switch char {
		case "\n":
			break charLoop
		case "\x7F":
			command = backspace(command)
			//println("backspace")
		default:
			fmt.Print(char)
			command = command + string(char)
		}

	}
	return command
}

func handleReader(reader *bufio.Reader) {
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		fmt.Print(str)
	}
}

func executeCommand(command string) {
	resetStdin()

	commandUnix := filepath.ToSlash(command)
	commandWords, err := shellquote.Split(commandUnix)
	if err != nil {
		fmt.Println("invalid qouting")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // cleanup resources eventually

	// Create the command with our context
	execCmd := exec.CommandContext(ctx, commandWords[0], commandWords[1:]...)
	execCmd.Env = os.Environ() // TODO: check
	execCmd.Stdin = os.Stdin

	// initialize stdout and stderr before start
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		fmt.Printf("unable to initialize process stdout")
	}
	defer stdoutPipe.Close()
	stdoutReader := bufio.NewReader(stdoutPipe)

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		fmt.Printf("unable to initialize process stdout")
	}
	defer stderrPipe.Close()
	stderrReader := bufio.NewReader(stderrPipe)

	// finally start the process!
	err = execCmd.Start()
	if err != nil {
		switch err.(type) {
		//   linux        , windows
		case *fs.PathError, *exec.Error:
			//fmt.Println(err.Error())
			fmt.Println("invalid executable")
		default:
			cobra.CheckErr(err)
		}
	}

	go handleReader(stdoutReader)
	go handleReader(stderrReader)

	pid := execCmd.Process.Pid
	//println("process started with pid : " + pid)
	_ = pid

	// finished flag become true
	// and check for any error
	executeErr := execCmd.Wait()

	if executeErr != nil {
		fmt.Println(executeErr.Error()) // TODO: handle in a better way
	}
}

func main() {
	stdin := make(chan string, 1)
	go readStdin(stdin)
	defer resetStdin()

	disableCtrlC()

commandLoop:
	for {
		print(prompt)
		command := readCommand(stdin)

		switch command {

		case "":
			break
		case "exit":
			break commandLoop
		case "clear":
			fmt.Print("\x1b\x5b\x48\x1b\x5b\x32\x4a")
		default:
			fmt.Println("\nrunning : " + command)
			executeCommand(command)
		}

		fmt.Println("")
	}
	close(stdin)

}
