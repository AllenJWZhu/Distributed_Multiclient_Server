# folder name of the package of interest
PKGNAME = gameServer

.PHONY: build final checkpoint all clean docs
.SILENT: build final checkpoint all clean docs

# compile the gameServer.
build:
	cd src/$(PKGNAME); go build gameServer.go game.go player.go messages.go

# run conformance tests.
final: build
	cd src/$(PKGNAME); go test -v -run Final

checkpoint: build
	cd src/$(PKGNAME); go test -v -run Checkpoint

all: build
	cd src/$(PKGNAME); go test -v
    
# delete all class files and docs, leaving only source
clean:
	rm -rf src/$(PKGNAME)/$(PKGNAME) src/$(PKGNAME)/$(PKGNAME)-doc.txt

# generate documentation for the package of interest
docs:
	cd src/$(PKGNAME); go doc -all > $(PKGNAME)-doc.txt
    
