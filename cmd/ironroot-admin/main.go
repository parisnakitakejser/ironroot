package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ironroot/ironroot/internal/cli/admin"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := admin.New().ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
