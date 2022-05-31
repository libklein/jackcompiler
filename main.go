package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func removeExtension(filePath string) string {
	extension := filepath.Ext(filePath)
	return filePath[:len(filePath)-len(extension)]
}

func getClassName(filePath string) string {
	return removeExtension(filepath.Base(filePath))
}

func getOutputPath(filePath string) string {
	return removeExtension(filePath) + ".vm"
}

func compileFile(r io.Reader, w io.Writer) {
	tokenizer := NewTokenizer(r)
	writer := NewVMWriter(w)

	compiler := NewJackCompiler(&tokenizer, &writer)
	compiler.Compile()
}

func processFile(path string) (outputPath string, err error) {
	// Open file for reading
	handle, openErr := os.Open(path)
	if openErr != nil {
		return "", fmt.Errorf("Could not open file %q for reading: %v", path, err)
	}
	defer handle.Close()

	// Open file for writing
	outputPath = getOutputPath(path)
	output, openErr := os.OpenFile(outputPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return outputPath, fmt.Errorf("Could not open output file %q for writing: %v", outputPath, err)
	}
	defer output.Close()

	// Translate
	compileFile(handle, output)

	return outputPath, nil
}

func collectFiles(fileOrDir string) (files []string, err error) {

	fileOrDirStat, err := os.Stat(fileOrDir)
	if err != nil {
		err = fmt.Errorf("Cannot stat file/dir %q\n", fileOrDir)
		return
	}

	// Collect files
	if fileOrDirStat.IsDir() {
		dirEntrys, readDirErr := os.ReadDir(fileOrDir)
		if readDirErr != nil {
			err = fmt.Errorf("Could not open directory %q!\n", fileOrDir)
			return
		}

		for _, dir := range dirEntrys {
			files = append(files, filepath.Join(fileOrDir, dir.Name()))
		}
	} else {
		files = []string{fileOrDir}
	}
	return
}

func main() {
	filename := flag.String("d", "", ".jack file to compile or directory containing .jack files")

	flag.Parse()

	if *filename == "" {
		flag.Usage()
		return
	}

	files, err := collectFiles(*filename)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, file := range files {
		if filepath.Ext(file) != ".jack" {
			continue
		}
		fmt.Printf("Compiling file %q\n", file)
		outputPath, err := processFile(file)
		if err != nil {
			fmt.Printf("Failed to compile %q: %s\n", file, err)
		}
		fmt.Printf("Saved as %q\n", outputPath)
	}
}
