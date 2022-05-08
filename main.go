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

func compileFile(r io.Reader, w io.Writer) {
	tokenizer := NewTokenizer(r)
	writer := NewVMWriter(w)

	compiler := NewJackCompiler(&tokenizer, &writer)
	compiler.Compile()
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
		// TODO
		//defer handle.Close()

		//dumpTokens(handle)
		//handle.Seek(0, 0)

		// Open file for writing
		outputPath := file[:len(file)-5] + ".vm"
		output, err := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			fmt.Printf("Could not generate assembly file %q: %v\n", outputPath, err)
		}
		// TODO
		//defer output.Close()

		// Translate
		compileFile(handle, output)
		/*if err != nil {
			fmt.Printf("Failed to compile file %q: %v\n", file, err)
		}*/
	}
}
