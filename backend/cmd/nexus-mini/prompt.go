package main

import (
	"bufio"
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/term"
)

func prompt(in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// promptPassword — терминал бол нууцалж уншина, pipe бол энгийнээр.
func promptPassword(in *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		return string(b), err
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
