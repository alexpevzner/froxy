########################################################################
# Environment
########################################################################

TOPDIR 	:= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
GOFILES	:= $(wildcard *.go)

########################################################################
# Tools
########################################################################

GO_ENV		:= CGO_ENABLED=1

GO_PATH 	:= $(TOPDIR)
ifneq		($(GOPATH),)
	GO_PATH	:= $(TOPDIR):$(GOPATH)
endif

########################################################################
# Cross-compilation support
#=======================================================================

FROXY	:= froxy
GO_CMD	:= go

ifneq 	($(GOOS)-$(GOARCH),-)
	GO_CMD := GOOS=$(GOOS) GOARCH=$(GOARCH) go
endif

ifeq 	($(GOOS)-$(GOARCH),windows-386)
	FROXY := froxy32.exe
	GO_CMD := GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go
endif

ifeq	($(GOOS)-$(GOARCH),windows-amd64)
	FROXY := froxy64.exe
	GO_CMD := GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go
endif

########################################################################
# Common targets
########################################################################

.PHONY:	all
.PHONY:	clean
.PHONY:	test
.PHONY:	vet
.PHONY:	tags

all:	subdirs_all do_all
test:	subdirs_test do_test
vet:	subdirs_vet do_vet
clean:	subdirs_clean do_clean

ifneq	($(GOFILES),)
do_all:
	$(GO_CMD) build -o $(FROXY)
else
do_test:
endif

ifneq	($(GOFILES),)
do_test:
	go test
else
do_test:
endif

ifneq	($(GOFILES),)
do_vet:
	go vet
else
do_vet:
endif

# This allows to OPTIONALLY define all_local
do_all:	all_local
all_local:

# This allows to OPTIONALLY define clean_local
do_clean:	clean_local
clean_local:

# tags target
tags:
	-cd $(TOPDIR); gotags -R . | grep -v '^!' > tags

# Automatic rebuilding of tags
ifeq	($(MAKELEVEL),0)
do_all do_test do_vet:    tags
endif

# Subdirs handling
subdirs_all subdirs_test subdirs_vet subdirs_clean:
	@for i in $(SUBDIRS); do \
		$(MAKE) -C $$i $(subst subdirs_,,$@) || exit 1; \
	done
