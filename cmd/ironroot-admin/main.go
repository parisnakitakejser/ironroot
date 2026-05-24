package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/parisnakitakejser/ironroot/internal/cli/admin"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := admin.New().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		var exitCoder interface{ ExitCode() int }
		if errors.As(err, &exitCoder) {
			os.Exit(exitCoder.ExitCode())
		}
		os.Exit(1)
	}
}
