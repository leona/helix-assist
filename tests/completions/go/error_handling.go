package main

import "os"

func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	<CURSOR>
	return data, nil
}

// Expected: completion should check if err != nil and handle it
