package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

const (
	prompt = ">>> "
)

func readStdin(out chan string, terminate chan bool) {
	//str, err := exec.Command("ls", "-gah").CombinedOutput()
	//if err != nil {
	//	println(err.Error())
	//}
	//fmt.Println(string(str))

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

	b := make([]byte, 1)
	for {
		select {
		case <-terminate:
			return
		default:
			os.Stdin.Read(b)
			//println(b[0])
			out <- string(b)
		}
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
			println("backspace")
		default:
			fmt.Print(char)
			command = command + string(char)
		}

	}
	return command
}

func main() {
	stdin := make(chan string, 1)
	terminate := make(chan bool, 1)
	go readStdin(stdin, terminate)
	defer resetStdin()

	disableCtrlC()

commandLoop:
	for {
		print(prompt)
		command := readCommand(stdin)

		switch command {
		case "exit":
			break commandLoop

		case "clear":
			println("\n\n\n\n\n\n\n\n\n")
		default:

		}

		//if char == "q" {
		//	terminate <- true
		//	break inpLoop
		//}

		fmt.Println("")
		fmt.Println("running : " + command)
		_ = command
	}
	close(stdin)

}
