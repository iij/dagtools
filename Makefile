MAKE ?= make
GO ?= go

APP_NAME := dagtools
APP_VERSION ?= 1.6.0
APP_RELEASE ?= 1
APP_PACKAGE ?= github.com/iij/dagtools

# amd64, 386
ARCH_TYPE ?= amd64
# ARCH_TYPE ?= 386

BUILD_DIR ?= build
BUILD_BIN_DIR := $(BUILD_DIR)/$(OS_TYPE)_$(ARCH_TYPE)

ifneq ($(wildcard $(WINDOWS)/system32/krnl32.dll),)
	LOCAL_OS_TYPE := windows
endif
ifneq ($(wildcard /sbin/bsdlabel),)
	LOCAL_OS_TYPE := freebsd
endif
ifneq ($(wildcard /System/Library/Extensions/AppleFileSystemDriver.kext),)
	LOCAL_OS_TYPE := darwin
endif
ifneq ($(wildcard /sbin/modprobe),)
	LOCAL_OS_TYPE := linux
endif

# windows, linux, darwin, freebsd, netbsd, dragonfly solaris, plan9
OS_TYPE ?= $(LOCAL_OS_TYPE)
# OS_TYPE ?= windows

BUILD_FILE_NAME := $(APP_NAME)
BUILD_BIN := $(BUILD_BIN_DIR)/$(APP_NAME)
BUILD_BIN_SUFFIX :=
ifeq ($(OS_TYPE),windows)
	BUILD_BIN_SUFFIX := .exe
	BUILD_FILE_NAME := $(APP_NAME)$(BUILD_BIN_SUFFIX)
	BUILD_BIN := $(BUILD_DIR)/$(OS_TYPE)_$(ARCH_TYPE)/$(BUILD_FILE_NAME)
endif

PACKAGE_NAME := $(APP_NAME)-$(OS_TYPE)-$(ARCH_TYPE)-$(APP_VERSION)
ARCHIVE_TYPE := tar.gz

ifeq ($(OS_TYPE),windows)
	ARCHIVE_TYPE := zip
endif

all: prepare $(BUILD_DIR)/$(PACKAGE_NAME).$(ARCHIVE_TYPE)

rpm: all rpmbuild

prepare: core/version.go

core/version.go:
	sed -e "s,__VERSION__,$(APP_VERSION),g" env/version.go.in > env/version.go

$(BUILD_DIR)/$(PACKAGE_NAME).tar.gz: $(BUILD_DIR)/$(PACKAGE_NAME)
	tar -zcf $(BUILD_DIR)/$(PACKAGE_NAME).tar.gz -C $(BUILD_DIR) $(PACKAGE_NAME)

$(BUILD_DIR)/$(PACKAGE_NAME).zip: $(BUILD_DIR)/$(PACKAGE_NAME)
	cd $(BUILD_DIR); zip -r $(PACKAGE_NAME).zip $(PACKAGE_NAME)

$(BUILD_DIR)/$(PACKAGE_NAME): $(BUILD_BIN)
	mkdir -p $@
	install -m 755 $< $@/
	install -m 644 ./dagtools.ini.sample $@/
	install -m 644 ./README.rst $@/
	install -m 644 ./CHANGELOG.rst $@/
	install -m 644 ./LICENSE.txt $@/

$(BUILD_BIN):
	mkdir -p $(dir $@)
	GOOS=$(OS_TYPE) GOARCH=$(ARCH_TYPE) $(GO) build -x -ldflags '-s -w' -o $@ $(APP_PACKAGE)

rpmbuild: $(BUILD_DIR)/$(APP_NAME)-linux-amd64-$(APP_VERSION)/$(APP_NAME)
	mkdir -p $(BUILD_DIR)/rpm/{SOURCES,RPMS,SRPMS,BUILD,BUILDROOT,SPECS,INSTALL}
	sed -e "s,__VERSION__,$(APP_VERSION),g" -e "s,__RELEASE__,$(APP_RELEASE),g" -e "s,__PACKAGE_NAME__,$(PACKAGE_NAME),g" $(APP_NAME).spec.in > $(BUILD_DIR)/$(APP_NAME).spec
	rpmbuild -bb --nodeps -D "_sourcedir $(PWD)/$(BUILD_DIR)" -D "_topdir $(PWD)/$(BUILD_DIR)/rpm" $(BUILD_DIR)/$(APP_NAME).spec

clean:
	rm -fr build/*
	rm -f core/version.go

install: prepare
	$(GO) install $(APP_PACKAGE)

test:
	$(GO) test $(APP_PACKAGE)/...
