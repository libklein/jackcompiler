package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func getClassName(filePath string) string {
	basename := filepath.Base(filePath)
	return basename[:len(basename)-5]
}

func dumpTokens(r io.Reader) {
	tokenizer := NewTokenizer(r)
	for tokenizer.Scan() {
		token := tokenizer.Token()
		fmt.Println(token)
	}
}

func compileFile(functionName string, r io.Reader) ([]string, error) {
	tokenizer := NewTokenizer(r)

	return CompileCode(&tokenizer)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s filename.jack\n", os.Args[0])
		return
	}

	fileOrDir := os.Args[1]
	fileOrDirStat, err := os.Stat(fileOrDir)
	if err != nil {
		fmt.Printf("Cannot stat file/dir %q\n", fileOrDir)
	}

	var (
		files []string
		//outputPath string
	)

	if fileOrDirStat.IsDir() {
		dirEntrys, err := os.ReadDir(fileOrDir)
		if err != nil {
			fmt.Printf("Could not open directory %q!\n", fileOrDir)
		}

		for _, dir := range dirEntrys {
			files = append(files, filepath.Join(fileOrDir, dir.Name()))
		}
	} else {
		files = []string{fileOrDir}
	}

	// Open file for writing
	//output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	//if err != nil {
	//	fmt.Printf("Could not generate assembly file %q: %v\n", outputPath, err)
	//}

	for _, file := range files {
		if filepath.Ext(file) != ".jack" {
			continue
		}

		// Open file for reading
		handle, err := os.Open(file)
		if err != nil {
			fmt.Printf("Could not open file %q: %v\n", file, err)
			return
		}

		dumpTokens(handle)
		handle.Seek(0, 0)

		// Translate
		vmCode, err := compileFile(getClassName(file), handle)
		if err != nil {
			fmt.Printf("Failed to compile file %q: %v\n", file, err)
		}

		// Close read file
		handle.Close()

		// Write assembly
		for _, line := range vmCode {
			fmt.Println(line)
			// output.WriteString(line + "\n")
		}
	}

	// Close output
	//output.Close()
}
