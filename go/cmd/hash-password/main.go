// Command hash-password prints an argon2id PHC hash for a plaintext password, used to
// generate the seed password hashes. Usage: go run ./cmd/hash-password <password>
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/henrique-yda/teste-tecnico-itau/internal/auth"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("usage: hash-password <password>")
	}
	h, err := auth.HashPassword(os.Args[1])
	if err != nil {
		log.Fatalf("hash-password: %v", err)
	}
	fmt.Println(h)
}
