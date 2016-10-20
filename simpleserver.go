// Licensed under GPL-2.0
package main

import (
	"flag"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

/*
#cgo LDFLAGS: -lcap
#include <sys/capability.h>
#include <errno.h>

static int dropAllCaps(void) {
    cap_t state;

    state = cap_init();
    if (!state) {
        cap_free(state);
    }


    if (cap_clear(state) < 0) {
        cap_free(state);
        return errno;
    }

    if (cap_set_proc(state) == -1) {
        cap_free(state);
        return errno;
    }

    cap_free(state);
    return 0;
}
*/
import "C"

var requestLog *log.Logger
var allowUploads *bool

func dropAllCaps() (err error) {
	errno := C.dropAllCaps()
	if errno != 0 {
		return syscall.Errno(errno)
	}
	return
}

func reqHandler(w http.ResponseWriter, r *http.Request) {
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

	requestLog.Println(r.RemoteAddr, fmt.Sprintf("\"%s %s %s\"", r.Method, r.URL, r.Proto))
	if r.Method == "GET" && statInfo.IsDir() {
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

	} else if r.Method == "GET" {
		mimeType := mime.TypeByExtension(filepath.Ext(filePath))
		if mimeType == "" {
			mimeType = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mimeType)
		w.Header().Set("Content-Length", strconv.FormatInt(statInfo.Size(), 10))

		file, err := os.Open(filePath)
		if err != nil {
			log.Println(err)
			return
		}
		defer func() {
			err := file.Close()
			if err != nil {
				panic(err)
			}
		}()

		_, err = io.Copy(w, file)
		if err != nil {
			log.Println(err)
			return
		}

	} else if r.Method == "POST" {
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
	} else {
		http.Error(w, "Unhandled request", http.StatusBadRequest)
	}
}

func main() {
	requestLog = log.New(os.Stdout,
		"REQ: ",
		log.Ldate|log.Ltime)

	err := syscall.Chroot(".")
	if err != nil {
		panic(err)
	}
	err = dropAllCaps()
	if err != nil {
		panic(err)
	}

	allowUploads = flag.Bool("allow-uploads", false, "Allow uploading of files")
	listenPort := flag.Int("port", 8000, "Listen on port (default 8000)")
	flag.Parse()

	http.HandleFunc("/", reqHandler)
	fmt.Println("Listening on port", *listenPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *listenPort), nil)
	if err != nil {
		panic(err)
	}
}
