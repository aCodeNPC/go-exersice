package textfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func ReadlineByScan(path string) {
	f, err := os.Open(path)
	if err != nil {
		em := fmt.Sprintf("ReadlineByScan open %s fail:%s", path, err.Error())
		fmt.Println(em)
		panic(em)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		num, err1 := strconv.Atoi(line)
		fmt.Println("ReadlineByScan read:", line, num, err1)
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("ReadlineByScan scanner fail", err)
	}
}

func ReadlineByBufioV1(path string) {
	F, err := os.Open(path)
	if err != nil {
		em := fmt.Sprintf("ReadlineByBufioV1 open %s fail:%s", path, err.Error())
		fmt.Println(em)
		panic(em)
	}
	defer F.Close()

	reader := bufio.NewReader(F)
	for {
		// ReadString reads until the first occurrence of delim in the input,
		// returning a string containing the data up to and including the delimiter.
		line, err := reader.ReadString('\n') // BUG: 换行符的不兼容性
		lines := strings.Split(line, "\n")
		line = lines[0]
		num, err1 := strconv.Atoi(line)
		fmt.Println("ReadlineByBufioV1 read:", line, num, err1)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ReadlineByBufioV1 fail", err)
		}
	}
}

// ReadLinesV3 reads all lines of the file.
func ReadlineByBufioV2(path string) {
	f, err := os.Open(path)
	if err != nil {
		em := fmt.Sprintf("ReadlineByBufioV1 open %s fail:%s", path, err.Error())
		fmt.Println(em)
		panic(em)
	}
	defer f.Close()

	r := bufio.NewReader(f)
	for {
		// ReadLine is a low-level line-reading primitive.
		// Most callers should use ReadBytes('\n') or ReadString('\n') instead or use a Scanner.
		bytes, _, err := r.ReadLine()
		lines := strings.Split(string(bytes), "\n")
		line := lines[0]
		num, err1 := strconv.Atoi(line)
		fmt.Println("ReadlineByBufioV2 read:", line, num, err1)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("ReadlineByBufioV2 fail", line, num, err)
		}
	}

}
