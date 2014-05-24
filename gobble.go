// Copyright 2014 Markus Dittrich. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// gobble is a simple program for retrieving files via
// http, https, and ftp á la wget

package main

import (
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
	toStdout    = flag.Bool("s", false, "output to stdout")
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
	url := normalizeURLTarget(*urlTarget)

	// start http client
	client := &http.Client{}
	resp, err := client.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// open output file; nil if stdout was requested
	file := os.Stdout
	if !*toStdout {
		file, err = openOutfile(*outFileName, url)
		if err != nil {
			log.Fatal("failed to open output file: ", err)
		}
		defer file.Close()
		printInfo(url, resp)
	}

	totalBytes := resp.ContentLength
	bytesRead, err := copyContent(resp.Body, file, totalBytes, *toStdout)
	if err != nil {
		log.Fatal(err)
	}

	if !*toStdout {
		fmt.Println(statusString(bytesRead, totalBytes, true))
	}
}

// copyContent reads the body content from the http connection and then
// copies it either to the provided file or stdou
func copyContent(body io.ReadCloser, file *os.File, totalBytes int64,
	wantStdout bool) (int, error) {

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
			return 0, fmt.Errorf("% bytes read but %d byte written", n, nOut)
		}

		bytesRead += n
		if !wantStdout {
			fmt.Print(statusString(bytesRead, totalBytes, false))
		}
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
	n, err := file.Write(content)
	if err != nil {
		return n, err
	}
	return n, nil
}

// openOutfile opens the output file if one was requested
// Otherwise, we assume the output file is index.html
func openOutfile(outFileName, urlTarget string) (*os.File, error) {

	fileName := outFileName
	if fileName == "" {

		// can we extract a
		urlInfo, err := url.Parse(urlTarget)
		if err != nil {
			return nil, err
		}
		if fileName = filepath.Base(urlInfo.Path); fileName == "." || fileName == "/" {
			fileName = "index.html"
		}
	}

	// if fileName already exists we bail
	if _, err := os.Stat(fileName); err == nil {
		return nil, fmt.Errorf("%s already exists\n", fileName)
	}

	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// normalizeURLTarget currently only checks if an URL starts with
// http:// and if not appends it
func normalizeURLTarget(urlTarget string) string {
	outString := urlTarget
	if !strings.HasPrefix(urlTarget, "http://") {
		outString = "http://" + urlTarget
	}
	return outString
}

// statusString returns the status string corresponding to the given
// number of bytes read.
// NOTE: Sites which don't provide the content length return a value of
// -1 for totalbytes. In this case we print a simpler content string
func statusString(bytesRead int, totalBytes int64, allDone bool) string {
	var msg string
	if allDone {
		msg = "Finished:    "
	} else {
		msg = "In progress: "
	}
	var formatString string
	if totalBytes == -1 {
		progressString := "<=>"
		formatString = fmt.Sprintf("%s %10d Bytes    %-30s  \r", msg, bytesRead,
			progressString)
	} else {
		percentage := float64(bytesRead) / float64(totalBytes) * 100
		progressString := strings.Join(
			[]string{progressBar[1 : 2+int(percentage/4)], ">"}, "")
		formatString = fmt.Sprintf("%s %10d Bytes    %-30s  %2.1f%%\r", msg,
			bytesRead, progressString, percentage)
	}
	return formatString
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
