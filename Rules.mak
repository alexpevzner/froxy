########################################################################
# Environment
########################################################################

TOPDIR 	:= $(dir $(abspath $(lastword $(MAKEFILE_LIST))))
GOFILES	:= $(wildcard *.go)

########################################################################
# Tools
########################################################################

GO_PKGDIR	:= -pkgdir=$(TOPDIR)/pkg/$(GOOS)_$(GOARCH)

GO_ENV		:= CGO_ENABLED=1
GO_ENV		+= CGO_CFLAGS="-I $(TOPDIR)/src"

GO_PATH 	:= $(TOPDIR)
ifneq		($(GOPATH),)
	GO_PATH	:= $(TOPDIR):$(GOPATH)
endif

BLD		= bld

GO_CMD		= GOPATH=$(GO_PATH) $(GO_ENV) go
GO_HOST		= $(shell go env GOHOSTOS)_$(shell go env GOARCH)
GO_TEST		= LD_LIBRARY_PATH=$(TOPDIR)/bin/$(GO_HOST) $(GO_CMD) test
GO_VET		= LD_LIBRARY_PATH=$(TOPDIR)/bin/$(GO_HOST) $(GO_CMD) vet

########################################################################
# Common targets
########################################################################

.PHONY:	all
.PHONY:	clean
.PHONY:	test
.PHONY:	vet
.PHONY:	tags
.PHONY:	tools

all:	do_all
test:	do_test
vet:	do_vet
clean:	do_clean

all test vet clean:
	@for i in $(SUBDIRS); do \
		$(MAKE) -C $$i $@ || exit 1; \
	done

ifneq	($(BOOTSTRAP),y)
do_all:
	$(BLD)
endif

ifneq	($(GOFILES),)
do_test:
	$(GO_TEST)
else
do_test:
endif

ifneq	($(GOFILES),)
do_vet:
	$(GO_VET)
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
	cd $(TOPDIR); gotags -R . | grep -v '^!' > tags; ctags -a -R

# Tools target
tools:

# Automatic rebuilding of tags and tools
ifeq	($(MAKELEVEL),0)
do_all do_test do_vet:    tags tools
endif

