package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/cli/client"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := client.New().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
