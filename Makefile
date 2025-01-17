DISTFILE=qd
BUILD_VERSION=`git describe --tags`

clean:
	rm -r dist

build:
	mkdir -p dist
	go build -ldflags "-X main.buildVersion=${BUILD_VERSION}" -o dist/${DISTFILE} -buildvcs=false

install:
	install -DZs dist/${DISTFILE} ${DESTDIR}/usr/bin