package main

import "os"

func tmpFile() *os.File {
	tmpdir := os.TempDir()
	file, err := os.Create(tmpdir + "/" + randSeq(10))
	if err != nil {
		panic(err)
	}
	return file
}
