SOURCES := $(shell find src -name '*.go' -type f)
all: gmailget

gmailget: $(SOURCES)
	wgo build gmailget
