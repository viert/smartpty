package main

import (
	"fmt"
	"github.com/viert/smartpty"
	"os"
	"os/exec"
	"regexp"
)

var (
	pwdExpr  = regexp.MustCompile(`[Pp]assword:`)
	echoExpr = regexp.MustCompile(`^[\n\r]+$`)
)

func main() {
	cmd := exec.Command("sudo", "bash")
	smart := smartpty.Create(cmd)

	// React on "Password:" expression once
	smart.Once(pwdExpr, func(data []byte, tty *os.File) []byte {
		// After the password response there will be an echo like \n\r
		// Let's skip it:
		smart.Once(echoExpr, func(data []byte, tty *os.File) []byte {
			// When echo comes, output nothing
			return []byte{}
		})
		// Send the password to the terminal
		tty.Write([]byte("MyR0otP@sswD\n"))
		// Remove the "Password:" from the chunk of data so
		// the user won't even notice she was prompted for passwd
		return pwdExpr.ReplaceAll(data, []byte{})
	})

	err := smart.Start()
	if err != nil {
		panic(err)
	}
	defer smart.Close()
	cmd.Wait()

	fmt.Println("Program exited")
}
