# Go parameters
GOCMD=go
GOMOD=${GOCMD} mod init go-go
GOTEST=${GOCMD} test
GORUN=${GOCMD} run main/*

mod:
	${GOMOD}

test:
	${GOTEST}

run:
	${GORUN}