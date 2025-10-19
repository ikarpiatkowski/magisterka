package main

import (
	"fmt"
	"log/slog"
	"math/rand"
	"os"
)

func random(min int, max int) int {
	return rand.Intn(max-min) + min
}

func annotate(err error, format string, args ...any) error {
	if err != nil {
		return fmt.Errorf("%s: %w", fmt.Sprintf(format, args...), err)
	}
	return nil
}

func fail(err error, format string, args ...any) {
	if err != nil {
		slog.Error(fmt.Sprintf("%s: %s", fmt.Sprintf(format, args...), err))
		os.Exit(1)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func genString(n int) string {
	b := make([]rune, n)

	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}

	return string(b)
}
