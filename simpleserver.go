// Licensed under GPL-2.0
package main

import (
	"bufio"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var (
	RequestLog *log.Logger
)

func reqHandler(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	filePath := filepath.Join(cwd, r.URL.Path)
	filePath, fpErr := filepath.EvalSymlinks(filePath)
	if fpErr != nil {
		log.Println(fpErr)
		return
	}

	if strings.HasPrefix(filePath, cwd) == false {
		log.Println("Trying to access dir outside of cwd")
		return
	}

	statInfo, statErr := os.Stat(filePath)
	if statErr != nil {
		log.Println(statErr)
		return
	}

	RequestLog.Println(r.RemoteAddr, fmt.Sprintf("\"%s %s %s\"", r.Method, r.URL, r.Proto))
	if statInfo.IsDir() {
		files, readErr := ioutil.ReadDir(filePath)
		if readErr != nil {
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
			fmt.Sprintf("<h1>%s</h1>\n<hr />\n", dirlistTitle)+
			"<ul>\n")

		listFmt := "\t<li><a href=\"%s\">..</a></li>\n"
		fmt.Fprintf(w, listFmt, filepath.Dir(r.URL.Path))
		for _, f := range files {
			fFull := html.EscapeString(filepath.Join(r.URL.Path, f.Name()))
			fmt.Fprintf(w, "\t<li><a href=\"%s\">%s</a></li>\n", fFull, f.Name())
		}

		fmt.Fprintf(w, "</ul>\n<hr />\n</body>\n</html>")

	} else {
		mimeType := mime.TypeByExtension(filepath.Ext(filePath))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)

		file, err := os.Open(filePath)
		if err != nil {
			log.Println(err)
			return
		}

		r := bufio.NewReader(file)
		buf := make([]byte, 1024)
		for {
			l, err := r.Read(buf)
			if err != nil && err != io.EOF {
				panic(err)
			}

			if l == 0 {
				break
			}

			if _, err := w.Write(buf[:l]); err != nil {
				panic(err)
			}
		}
	}
}

func main() {
	RequestLog = log.New(os.Stdout,
		"REQ: ",
		log.Ldate|log.Ltime)

	http.HandleFunc("/", reqHandler)
	http.ListenAndServe(":8000", nil)
}
