prefix=/usr/local

all:
	go build -o simpleserver simpleserver.go

install:
	install -m 0755 simpleserver $(prefix)/bin
	setcap cap_sys_chroot+ep $(prefix)/bin/simpleserver

clean:
	if [ -e simpleserver ]; then rm simpleserver; fi
