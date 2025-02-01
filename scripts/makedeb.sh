#!/bin/bash

if [ -z "$VERSION" ]; then
    echo "VERSION not set, exiting"
    exit 1
fi

if [ -z "$BUILD_NUMBER" ]; then
    echo "BUILD_NUMBER not set, exiting"
    exit 1
fi

DEBIAN_OUT_DIR=./deb.output
if [ ! -d $DEBIAN_OUT_DIR ] ; then
    mkdir $DEBIAN_OUT_DIR
fi
rm -f $DEBIAN_OUT_DIR/*

FULL_VERSION=$VERSION-$BUILD_NUMBER
export DEBEMAIL=stephen.c.sanders@gmail.com
export DEBFULLNAME="stephen.c.sanders@gmail.com"

rm -f  debian/changelog debian/changelog.dch

dch --create \
--distribution stable \
--package "link-share" \
--newversion $FULL_VERSION \
"Build for release"

# debuild drops most PATH elements, prepend-path allows us to add back.
# TODO: $HOME/go/bin probably not the right thing to do here.
debuild  --preserve-envvar=VERSION \
    --preserve-envvar=BUILD_NUMBER \
    --prepend-path=/opt/go/bin:$HOME/go/bin \
    --no-lintian -i -us -uc -b -tc

#
# debuild is really keen on using ../ as an output. Easier to collect result
# than force configuration
# https://www.patreon.com/posts/building-debian-23177439 is close but I was
# unable to get anything other than the build to work properly.
#
mv ../link-share_*  $DEBIAN_OUT_DIR

rm -rf debian/debhelper-build-stamp \
debian/files \
debian/link-share.substvars \
debian/link-share \
debian/.debhelper

mkdir -p build-output
cp $DEBIAN_OUT_DIR/link-share_$VERSION-$BUILD_NUMBER_*.deb ./build-output/

rm -rf $DEBIAN_OUT_DIR
