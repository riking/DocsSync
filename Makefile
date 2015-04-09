
SRCS=sync.go
ALLBINS=DocsSync_x86 DocsSync_x64 DocsSync_x86.exe DocsSync_x64.exe

DocsSync_all.zip: $(ALLBINS)
	zip $@ $(ALLBINS)

DocsSync_x86: $(SRCS)
	GOOS=linux GOARCH=386 go build -o $@
DocsSync_x64: $(SRCS)
	GOOS=linux GOARCH=amd64 go build -o $@
DocsSync_x86.exe: $(SRCS)
	GOOS=windows GOARCH=386 go build -o $@
DocsSync_x64.exe: $(SRCS)
	GOOS=windows GOARCH=amd64 go build -o $@
