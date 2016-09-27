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

func req_handler(w http.ResponseWriter, r *http.Request) {
	cwd, _ := os.Getwd()
	file_path := filepath.Join(cwd, r.URL.Path)
	file_path, fp_err := filepath.EvalSymlinks(file_path)
	if fp_err != nil {
		log.Println(fp_err)
		return
	}

	if strings.HasPrefix(file_path, cwd) == false {
		log.Println("Trying to access dir outside of cwd")
		return
	}

	stat_info, stat_err := os.Stat(file_path)
	if stat_err != nil {
		log.Println(stat_err)
		return
	}

	RequestLog.Println(r.RemoteAddr, fmt.Sprintf("\"%s %s %s\"", r.Method, r.URL, r.Proto))
	if stat_info.IsDir() {
		files, read_err := ioutil.ReadDir(file_path)
		if read_err != nil {
			log.Println(read_err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		dirlist_title := fmt.Sprintf("Directory listing for %s", r.URL.Path)

		fmt.Fprintf(w, "<!DOCTYPE html>\n"+
			"<html>\n"+
			"<head>\n"+
			"<meta http-equiv=\"Content-Type\" content=\"text/html; charset=utf-8\">\n"+
			fmt.Sprintf("<title>%s</title>\n", dirlist_title)+
			"</head>\n<body>\n"+
			fmt.Sprintf("<h1>%s</h1>\n<hr />\n", dirlist_title)+
			"<ul>\n")

		list_fmt := "\t<li><a href=\"%s\">..</a></li>\n"
		fmt.Fprintf(w, list_fmt, filepath.Dir(r.URL.Path))
		for _, f := range files {
			f_full := html.EscapeString(filepath.Join(r.URL.Path, f.Name()))
			fmt.Fprintf(w, "\t<li><a href=\"%s\">%s</a></li>\n", f_full, f.Name())
		}

		fmt.Fprintf(w, "</ul>\n<hr />\n</body>\n</html>")

	} else {
		mime_type := mime.TypeByExtension(filepath.Ext(file_path))
		if mime_type == "" {
			mime_type = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mime_type)

		file, err := os.Open(file_path)
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

	http.HandleFunc("/", req_handler)
	http.ListenAndServe(":8000", nil)
}
