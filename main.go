// Copyright (C) 2017 Adam Hose adis@blad.is
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

var requestLog *log.Logger
var allowUploads *bool

func listDir(filePath string, w http.ResponseWriter, r *http.Request) {
	files, readErr := ioutil.ReadDir(filePath)
	if readErr != nil {
		http.Error(w, "Directory read error", http.StatusInternalServerError)
		log.Println(readErr)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	dirlistTitle := fmt.Sprintf("Directory listing for %s", r.URL.Path)

	fmt.Fprintf(w, "<!DOCTYPE html>\n"+
		"<html>\n"+
		"<head>\n"+
		"<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\n"+
		fmt.Sprintf("<title>%s</title>\n", dirlistTitle)+
		"</head>\n<body>\n"+
		fmt.Sprintf("<h1 style=\"display: inline-block;\">%s</h1>\n", dirlistTitle))
	if *allowUploads {
		fmt.Fprintf(w, fmt.Sprintf("<form action=\"%s\" method=\"post\" enctype=\"multipart/form-data\">\n", r.URL.Path)+
			"<label for=\"file\">Upload file: </label>\n"+
			"<input type=\"file\" name=\"file\">\n"+
			"<input type=\"submit\" value=\"Submit\" />\n</form>\n")
	}
	fmt.Fprintf(w, "<hr />\n<ul>\n")

	listFmt := "<li>\n<a href=\"%s\">..</a>\n</li>\n"
	fmt.Fprintf(w, listFmt, filepath.Dir(r.URL.Path))
	for _, f := range files {
		fFull := html.EscapeString(filepath.Join(r.URL.Path, f.Name()))
		fmt.Fprintf(w, "<li>\n<a href=\"%s\">%s</a>\n</li>\n", fFull, f.Name())
	}

	fmt.Fprintf(w, "</ul>\n<hr />\n</body>\n</html>")
}

func uploadFile(statInfo os.FileInfo, filePath string, w http.ResponseWriter, r *http.Request) {
	if !statInfo.IsDir() {
		http.Error(w, "Cannot upload to non-directory file", http.StatusForbidden)
		log.Println("Cannot upload to non-directory file")
		return
	}

	err := r.ParseMultipartForm(15485760)
	if err != nil {
		log.Println(err)
		return
	}
	formFile, handler, err := r.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer func() {
		err := formFile.Close()
		if err != nil {
			panic(err)
		}
	}()

	outFilePath := filepath.Join(filePath, handler.Filename)
	if _, err := os.Stat(outFilePath); err == nil {
		http.Error(w, "File already exists", http.StatusForbidden)
		log.Println("File already exists")
		return
	}
	f, err := os.OpenFile(outFilePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		http.Error(w, "Cannot write file", http.StatusForbidden)
		fmt.Println(err)
		return
	}
	defer func() {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}()

	_, err = io.Copy(f, formFile)
	if err != nil {
		log.Println(err)
		return
	}
	http.Redirect(w, r, r.URL.Path, 302)

}

func handleWrapper(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		requestLog.Println(r.RemoteAddr, fmt.Sprintf("\"%s %s %s\"", r.Method, r.URL, r.Proto))

		if !*allowUploads && r.Method == "POST" {
			http.Error(w, "Uploads not allowed", http.StatusForbidden)
			log.Println("Uploads not allowed")
			return
		}

		cwd, _ := os.Getwd()
		filePath := filepath.Join(cwd, r.URL.Path)
		if strings.HasPrefix(filePath, cwd) == false {
			log.Println("Trying to access dir outside of cwd")
			return
		}

		statInfo, statErr := os.Stat(filePath)
		if statErr != nil {
			http.NotFound(w, r)
			log.Println(statErr)
			return
		}

		if r.Method == "GET" && statInfo.IsDir() {
			listDir(filePath, w, r)
		} else if r.Method == "POST" {
			uploadFile(statInfo, filePath, w, r)
		} else {
			h.ServeHTTP(w, r)
		}
	}
}

func main() {
	requestLog = log.New(os.Stdout,
		"REQ: ",
		log.Ldate|log.Ltime)

	allowUploads = flag.Bool("allow-uploads", false, "Allow uploading of files")
	listenPort := flag.Int("port", 8000, "Listen on port (default 8000)")
	flag.Parse()

	err := syscall.Chroot(".")
	if err != nil {
		fmt.Println("Could not chroot, here be dragons")
	}

	l, err := net.Listen("tcp", fmt.Sprintf(":%d", *listenPort))
	if err != nil {
		panic(err)
	}

	// Drop POSIX capabilities, only works on linux
	err = DropAllCaps()
	if err != nil {
		panic(err)
	}

	http.Handle("/", handleWrapper(http.FileServer(http.Dir("./"))))
	http.Serve(l, nil)
}
