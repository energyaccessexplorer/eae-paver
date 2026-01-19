default: clean build

-include .env

export PAVER_SERVER := ${PAVER_SERVER}
export PAVER_SOCKET := ${PAVER_SOCKET}
export PAVER_WORKDIR := ${PAVER_WORKDIR}
export PAVER_USER := ${PAVER_USER}
export PAVER_CMD := paver \
	-server \
	-pubkey ${PAVER_PUBKEY} \
	-socket ${PAVER_SOCKET} \
	-buckets ${PAVER_BUCKETS}

run:
	-@ pkill -9 paver
	./${PAVER_CMD}

build:
	go get
	go fmt

	CGO_LDFLAGS="-L/usr/local/lib -lgdal" \
	CGO_CFLAGS="-I/usr/local/include -I/usr/include/gdal" \
	go build -ldflags "-s \
		-X main.SOCKET_ACCEPT_PATTERN=${PAVER_SOCKET_ACCEPT_PATTERN}"

	envsubst <paver.service-tmpl >paver.service
	cat paver.service

clean:
	-rm -f paver paver.service

install: build
	sudo install -o root -m 755 \
		paver \
		/usr/local/bin/

	sudo install -o root -g root -m 644 \
		paver.service \
		/etc/systemd/system/

deploy:
	ssh ${PAVER_SERVER} "cd ${PAVER_SRCDIR}; git stash; git pull; touch deploy.diff; patch -p1 <deploy.diff;"
	ssh ${PAVER_SERVER} "sudo systemctl stop paver.service"
	ssh ${PAVER_SERVER} "cd ${PAVER_SRCDIR}; make install;"
	ssh ${PAVER_SERVER} "sudo systemctl daemon-reload"
	ssh ${PAVER_SERVER} "sudo systemctl start paver.service"

all: clean build
