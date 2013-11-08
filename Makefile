all: linefan
	prove t

linefan: linefan.go
	go build linefan.go
