ROOT := ${CURDIR}

GCFLAGS='-N -l'
GCLDFLAGS=''
BUILD_CMD =go build \
                -gcflags=${GCFLAGS} \
                -ldflags=${GCLDFLAGS} \
                -o ${ROOT}/bin/$@ \
                ./cmd/$@

ifeq ($(shell test -e /etc/redhat-release && echo -n yes),yes)
	RPM_INSTALLED := 1
endif

all: generate link-share 

link-share:
	@echo "@ building: [$@]..."
	${BUILD_CMD}

# output will be found in ~/rpmbuild/RPM/x86_64
#
rpm:
	tar -czf ${HOME}/rpmbuild/SOURCES/link-share.tar.gz .
	rpmbuild -bb  \
		--define "build_number ${BUILD_NUMBER}" \
		--define "version ${VERSION}" \
		rpm-spec/link-share.spec
	mkdir -p ${PWD}/build-output
	mv ${HOME}/rpmbuild/RPMS/x86_64/link-share*.rpm build-output
	rm ${HOME}/rpmbuild/SOURCES/link-share.tar.gz

# Output in build-output.
#
deb:
	BUILD_NUMBER=${BUILD_NUMBER} VERSION=${VERSION} scripts/makedeb.sh

# used by debian package create.
install:
	mkdir -p $(DESTDIR)$(prefix)/etc
	mkdir -p $(DESTDIR)$(prefix)/bin
	install bin/link-share $(DESTDIR)$(prefix)/bin
	cp scripts/link-share.sh $(DESTDIR)$(prefix)/bin
	chmod 755 $(DESTDIR)$(prefix)/bin/link-share.sh
	cp etc/link-share.service $(DESTDIR)$(prefix)/etc

generate : link_proto/link-share.pb.go

link_proto/link-share.pb.go : link_proto/link-share.proto
	protoc -I link_proto \
	--go_out=link_proto \
	--go_opt=paths=source_relative \
	link_proto/*.proto

#
# alma 10 requires enabling /etc/yum.repos.d/almalinux-crb.repo to install
# protobuf-compiler.
#
depends:
ifdef RPM_INSTALLED
	sudo dnf install protobuf-compiler
else
	sudo apt install protobuf-compiler
endif
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

clean:
	rm -f bin/*
	rm -f link_proto/*.pb.go
	rm -rf build-output/*

realclean: clean
	go clean -modcache -cache

module:
	go mod init github.com/code-ointment/link-share
