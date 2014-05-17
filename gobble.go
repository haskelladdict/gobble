// Copyright 2014 Markus Dittrich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// gobble is a simple program for retrieving files via
// http, https, and ftp á la wget

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
	"path/filepath"
	"strings"
)

// command line settings
var (
	urlTarget   = flag.String("u", "", "url to download")
	outFileName = flag.String("o", "", "name of output file")
	toStdOut    = flag.Bool("s", false, "output to stdout")
)

// general settings
var (
	numBytes = 40960 // chunk site for reading and writing
	version  = 0.1   // gobble version
)

// progress bar
var progressBar = "-----------------------------------"

func main() {

	flag.Parse()
	if *urlTarget == "" {
		usage()
	}

	// start http client
	client := &http.Client{}
	resp, err := client.Get(*urlTarget)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// open output file; nil if stdout was requested
	var file *os.File
	if !*toStdOut {
		file, _ = openOutfile()
		printInfo(*urlTarget, resp)
	}
	defer file.Close()

	totalBytes := resp.ContentLength
	if totalBytes == -1 {
		log.Fatal(errors.New("content has zero length"))
	}

	bytesRead, err := copyContent(resp.Body, file, totalBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(statusString(bytesRead, totalBytes))
}

// copyContent reads the body content from the http connection and then
// copies it either to the provided file or stdou
func copyContent(body io.ReadCloser, file *os.File, totalBytes int64) (int, error) {

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
		fmt.Print(statusString(bytesRead, totalBytes))
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
func openOutfile() (*os.File, error) {

	fileName := *outFileName
	if fileName == "" {
		urlInfo, err := url.Parse(*urlTarget)
		if err != nil {
			return nil, err
		}
		if fileName = filepath.Base(urlInfo.Path); fileName == "" {
			return nil, errors.New(fmt.Sprint("Could not extract filename from URL\n"))
		}
	}

	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// statusString returns the status string corresponding to the given
// number of bytes read.
func statusString(bytesRead int, totalBytes int64) string {
	percentage := float64(bytesRead) / float64(totalBytes) * 100
	progressString := strings.Join(
		[]string{progressBar[1 : 2+int(percentage/4)], ">"}, "")
	return fmt.Sprintf("progress: %10d Bytes    %-30s  %2.1f%%\r", bytesRead,
		progressString, percentage)
}

// printInfo prints a brief informative header about the connection
func printInfo(urlTarget string, resp *http.Response) {
	fmt.Println("********* This is gobble version ", version, " ***************")

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
