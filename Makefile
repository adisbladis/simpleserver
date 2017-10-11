PREFIX=/usr/local
LDFLAGS="-w -s"

all:
	go build -ldflags $(LDFLAGS) -o simpleserver

install:
	install -m 0755 simpleserver $(PREFIX)/bin
	setcap cap_sys_chroot+ep $(PREFIX)/bin/simpleserver

clean:
	if [ -e simpleserver ]; then rm simpleserver; fi
