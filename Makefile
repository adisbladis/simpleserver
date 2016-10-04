prefix=/usr/local

all:
	go build -o simpleserver simpleserver.go
	sudo setcap cap_sys_chroot+ep simpleserver

install:
	install -m 0755 simpleserver $(prefix)/bin

clean:
	if [ -e simpleserver ]; then rm simpleserver; fi
