GOCMD=go
GOBUILD=$(GOCMD) build
GOGET=$(GOCMD) get
CURRENT := $(shell pwd)
OUTPUTFILENAME := "main"
BUILDDIR=./build
DISTDIR=$(BUILDDIR)/dist
PKGDIR=$(BUILDDIR)/pkg
LDFLAGS := -X 'main.version=$(VERSION)'
GOXARCH := "386 amd64"
## GOXOUTPUT := "$(PKGDIR)/$(OUTPUTFILENAME)_{{.OS}}_{{.Arch}}/$(OUTPUTFILENAME)"
GOXOUTPUT := "$(PKGDIR)/$(NAME)_{{.OS}}_{{.Arch}}/{{.Dir}}"
VERSION := $(shell git describe --tags --abbrev=0)


.PHONY: test
test: lint gofmt


.PHONY: testdeps
testdeps:
	go get -d -v -t ./...
	GO111MODULE=off \
	go get golang.org/x/lint/golint \
		golang.org/x/tools/cmd/cover \
		github.com/axw/gocov/gocov \
		github.com/mattn/goveralls

LINT_RET = .golint.txt
.PHONY: lint
lint: testdeps
	go vet .
	rm -f $(LINT_RET)
	golint ./... | tee $(LINT_RET)
	test ! -s $(LINT_RET)

GOFMT_RET = .gofmt.txt
.PHONY: gofmt
gofmt: testdeps
	rm -f $(GOFMT_RET)
	gofmt -s -d *.go | tee $(GOFMT_RET)
	test ! -s $(GOFMT_RET)

.PHONY: cover
cover: testdeps
	goveralls


.PHONY: setup
## Install dependencies
setup:
	$(GOGET) github.com/mitchellh/gox
	$(GOGET) github.com/tcnksm/ghr
	$(GOGET) -d -t ./...

.PHONY: cross-build
## Cross build binaries
cross-build:
	rm -rf $(PKGDIR)
	gox -os=$(GOXOS) -arch=$(GOXARCH) -output=$(GOXOUTPUT)

.PHONY: package
## Make package
package: cross-build
	rm -rf $(DISTDIR)
	mkdir $(DISTDIR)
	pushd $(PKGDIR) > /dev/null && \
		for P in `ls | xargs basename`; do zip -r $(CURRENT)/$(DISTDIR)/$$P.zip $$P; done && \
		popd > /dev/null

.PHONY: release
## Release package to Github
release: package
	ghr -u yhidetoshi -r GoAWSDeleteAmisLaunchConfigsTool $(VERSION) $(DISTDIR)
