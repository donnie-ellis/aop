// genhash prints a bcrypt hash of the password given as the first argument,
// or "password" if none is provided. Used by make seed-user and smoke tests.
package main

import (
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	pw := "password"
	if len(os.Args) > 1 {
		pw = os.Args[1]
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(string(h))
}
