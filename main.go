package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
	"github.com/creack/pty"
	"github.com/kballard/go-shellquote"
	"golang.org/x/term"
)

const (
	prompt = "[Yourname@Your PC]$ "
)

func execute(command string) {
	commandUnix := filepath.ToSlash(command)

	commandWords, err := shellquote.Split(commandUnix)
	if err != nil {
		fmt.Println("invalid qouting")
	}

	execCmd := exec.Command(commandWords[0], commandWords[1:]...)
	execCmd.Env = os.Environ()

	ptmx, err := pty.Start(execCmd)
	if err != nil {
		log.Printf("cannot execute command, %s", err)
		return
	}

	defer func() { _ = ptmx.Close() }() // Best effort.

	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go func() { _, _ = io.Copy(ptmx, os.Stdin) }()
	_, _ = io.Copy(os.Stdout, ptmx)
}

// Function constructor - constructs new function for listing given directory
func listFiles(path string) func(string) []string {
	return func(line string) []string {
		names := make([]string, 0)
		files, _ := os.ReadDir(path)
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names
	}
}

func buildCompleter() *readline.PrefixCompleter {
	return readline.NewPrefixCompleter(
		readline.PcItem("exit"),
		readline.PcItem("clear"),
		readline.PcItem("setprompt"),
		readline.PcItem("cd", readline.PcItemDynamic(listFiles("./"))),
		readline.PcItem("vim", readline.PcItemDynamic(listFiles("./"))),
		readline.PcItem("nvim", readline.PcItemDynamic(listFiles("./"))),
	)
}

func filterInput(r rune) (rune, bool) {
	switch r {
	// block CtrlZ feature
	case readline.CharCtrlZ:
		return r, false
	}
	return r, true
}

func main() {
	l, err := readline.NewEx(&readline.Config{
		Prompt:          prompt,
		HistoryFile:     "/tmp/kitty-hist.tmp",
		AutoComplete:    buildCompleter(),
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",

		HistorySearchFold:   true,
		FuncFilterInputRune: filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer l.Close()
	l.CaptureExitSignal()

	log.SetOutput(l.Stderr())

	for {
		line, err := l.Readline()
		if errors.Is(err, readline.ErrInterrupt) {
			if len(line) == 0 {
				break
			}

			continue
		} else if errors.Is(err, io.EOF) {
			break
		}

		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "setprompt"):
			if len(line) <= len("setprompt") {
				log.Println("setprompt <prompt>")

				break
			}

			l.SetPrompt(line[len("setprompt"):] + " ")
		case line == "exit":
			goto exit
		case line == "clear":
			os.Stdout.Write([]byte("\x1b\x5b\x48\x1b\x5b\x32\x4a"))
		case line == "":
		default:
			execute(line)
		}
	}
exit:
}
