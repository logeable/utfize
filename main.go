package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

var (
	srcDir         string
	dstDir         string
	sourceEncoding string
	sourceFile     string
	verbose        bool
)

const (
	gbkEncoding       = "GBK"
	defSourceEncoding = gbkEncoding
)

var (
	encodingMap = map[string]encoding.Encoding{
		"GBK":     simplifiedchinese.GBK,
		"GB18030": simplifiedchinese.GB18030,
		"UTF8":    unicode.UTF8,
	}
)

func init() {
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal("get working directory failed:", err)
	}
	flag.StringVar(&srcDir, "s", wd, "source directory")
	flag.StringVar(&dstDir, "d", filepath.Join(wd, "output"), "destination directory")
	flag.StringVar(&sourceEncoding, "enc", defSourceEncoding, "source encoding: gbk")
	flag.StringVar(&sourceFile, "sf", "", "source file")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.Parse()

	srcDir, _ = filepath.Abs(srcDir)
	dstDir, _ = filepath.Abs(dstDir)
}

func main() {
	var err error
	if sourceFile != "" {
		err = transFile()
	} else {
		err = transDir()
	}
	if err != nil {
		log.Fatal("transform failed:", err)
	}
}

func transFile() error {
	verboseOutput("transform file mode")
	verboseOutput(fmt.Sprintf("src: %s [%s]", sourceFile, sourceEncoding))
	bs, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		return err
	}
	enc, ok := encodingMap[sourceEncoding]
	if !ok {
		return fmt.Errorf("invalid encoding: %s", sourceEncoding)
	}
	data, err := transformToUtf8(bs, enc)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func transDir() error {
	verboseOutput("transform directory mode")
	verboseOutput(fmt.Sprintf("src: %s dst: %s [%s]", srcDir, dstDir, sourceEncoding))
	fi, err := os.Stat(srcDir)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	os.RemoveAll(dstDir)
	err = os.Mkdir(dstDir, mode)
	if err != nil {
		return err
	}

	enc, ok := encodingMap[sourceEncoding]
	if !ok {
		return fmt.Errorf("invalid encoding: %s", enc)
	}

	var wg sync.WaitGroup
	defer wg.Wait()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		verboseOutput(path)
		newPath := strings.Replace(path, srcDir, dstDir, 1)
		if info.IsDir() {
			return os.MkdirAll(newPath, info.Mode())
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			bs, err := ioutil.ReadFile(path)
			if err != nil {
				fmt.Println(err)
				return
			}
			data, err := transformToUtf8(bs, enc)
			if err != nil {
				fmt.Println(err)
				return
			}
			err = ioutil.WriteFile(newPath, data, 0666)
			if err != nil {
				fmt.Println(err)
			}
		}()
		return nil
	})
}

func transformToUtf8(data []byte, enc encoding.Encoding) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(data), enc.NewDecoder())
	bs, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func verboseOutput(msg string) {
	if verbose {
		fmt.Println(msg)
	}
}
