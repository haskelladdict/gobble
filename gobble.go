package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
)

var (
	urlTarget = flag.String("u", "", "url to download")
	outFile   = flag.String("o", "", "name of output file")
	numBytes  = 40960 // chunk site for reading and writing
	version   = 0.1   // gobble version
)

func main() {

	flag.Parse()
	if *urlTarget == "" {
		usage()
	}

	// open output file if requested
	file := openOutfile()
	defer file.Close()

	// start http client
	client := &http.Client{}
	resp, err := client.Get(*urlTarget)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	printInfo(*urlTarget, resp)

	totalLength := resp.ContentLength
	if totalLength == -1 {
		log.Fatal(errors.New("content has zero length"))
	}

	bytesRead, err := copyContent(resp.Body, file, totalLength)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("progress: %10d Bytes   %2.1f%%\n", bytesRead,
		float64(bytesRead)/float64(totalLength)*100)
}

// copyContent reads the body content from the http connection and then
// copies it either to the provided file or stdou
func copyContent(body io.ReadCloser, file *os.File, totalLength int64) (int, error) {

	buffer := make([]byte, numBytes)
	bytesRead := 0
	n := 0
	for {
		// read numBytes
		var err error
		n, err = io.ReadFull(body, buffer)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break // this is the regular end-of-file - we are done
			} else {
				return 0, err
			}
		}

		// write numBytes
		nOut, err := bufWrite(buffer, file)
		if err != nil {
			log.Fatal(err)
		} else if nOut != n {
			return 0, errors.New(
				fmt.Sprintf("% bytes read but %d byte written", n, nOut))
		}

		bytesRead += n
		fmt.Printf("progress: %10d Bytes   %2.1f%%\r", bytesRead,
			float64(bytesRead)/float64(totalLength)*100)
	}

	// write whatever is left
	_, err := bufWrite(buffer[:n], file)
	if err != nil {
		return 0, err
	}

	bytesRead += n
	return bytesRead, nil
}

// bufWrite writes content either to stdout or the requested output file
func bufWrite(content []byte, file *os.File) (int, error) {
	var n int
	var err error
	if file == nil {
		if n, err = os.Stdout.Write(content); err != nil {
			return n, err
		}
	} else {
		if n, err = file.Write(content); err != nil {
			return n, err
		}
	}
	return n, nil
}

// openOutfile opens the output file if one was requested
// NOTE: If something goes wrong this function bails
func openOutfile() *os.File {
	if *outFile == "" {
		return nil
	}

	file, err := os.Create(*outFile)
	if err != nil {
		log.Fatal(err)
	}
	return file
}

func printInfo(urlTarget string, resp *http.Response) {
	fmt.Println("This is gobble version ", version)

	urlInfo, err := url.Parse(urlTarget)
	if err != nil {
		return
	}

	cname, _ := net.LookupCNAME(urlInfo.Host)
	ips, _ := net.LookupIP(cname)
	fmt.Println("Connecting to", cname, "  ", ips)
	fmt.Printf("Status %s   Protocol %s  TransferEncoding %v\n", resp.Status,
		resp.Proto, resp.TransferEncoding)
	fmt.Printf("Content Length: %d bytes\n", resp.ContentLength)
	fmt.Println()
}

// usage prints the package usage and then exits
func usage() {
	fmt.Println(os.Args[0], "[options]", "\n\noptions:")
	flag.PrintDefaults()
	os.Exit(1)
}
