ROOT := ${CURDIR}
GOPRIVATE := bitbucket.org,*.bitbucket.org
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

all: .initconfig generate link-share 

# Correct for github later.
.initconfig:
	git config --global url.ssh://git@bitbucket.org/.insteadOf \
		https://bitbucket.org/
	touch .initconfig

link-share:
	@echo "@ building: [$@]..."
	${BUILD_CMD}

generate : link_proto/link-share.pb.go

link_proto/link-share.pb.go : link_proto/link-share.proto
	protoc -I link_proto \
	--go_out=link_proto \
	--go_opt=paths=source_relative \
	link_proto/*.proto

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

realclean: clean
	go clean -modcache -cache
