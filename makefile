default: clean build

-include .env

PAVER_CMD = paver \
	-server \
	-pubkey ${PAVER_PUBKEY} \
	-socket ${PAVER_SOCKET}

run:
	-@pkill -9 paver
	./${PAVER_CMD}

build:
	go get
	go fmt

	CGO_LDFLAGS="-L/usr/local/lib -lgdal" \
	CGO_CFLAGS="-I/usr/local/include" \
	go build -ldflags "-s \
		-X main.SOCKET_ACCEPT_PATTERN=${PAVER_SOCKET_ACCEPT_PATTERN} \
		-X main.tmpdir=${PAVER_TMPDIR} \
		-X main.S3KEY=${PAVER_S3KEY} \
		-X main.S3SECRET=${PAVER_S3SECRET} \
		-X main.S3PROVIDER=${PAVER_S3PROVIDER} \
		-X main.S3BUCKET=${PAVER_S3BUCKET} \
		-X main.S3DIRECTORY=${PAVER_S3DIRECTORY} \
		-X main.S3ACL=${PAVER_S3ACL}"

.export PAVER_CMD
.export PAVER_SOCKET
.export PAVER_WORKSPACE
.export PAVER_USER
	@envsubst <paver.service-tmpl >paver.service

clean:
	-rm -f paver paver.service

install:
	bmake build

	sudo install -o root -m 755 \
		paver \
		/usr/local/bin/

	sudo install -o root -g root -m 644 \
		paver.service \
		/etc/systemd/system/

deploy:
	ssh ${PAVER_SERVER} "cd ${PAVER_WORKSPACE}; git stash; git pull; git stash pop;"
	ssh ${PAVER_SERVER} "sudo systemctl stop paver.service"
	ssh ${PAVER_SERVER} "cd ${PAVER_WORKSPACE}; bmake install;"
	ssh ${PAVER_SERVER} "sudo systemctl daemon-reload"
	ssh ${PAVER_SERVER} "sudo systemctl start paver.service"

all: clean build
