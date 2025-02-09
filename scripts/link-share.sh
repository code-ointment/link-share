#!/bin/bash

fullpath=$(realpath $0)
scriptpath=$(dirname $fullpath)
#
# dirname twice is trimming the path string. 
#
homepath=$(dirname $scriptpath)
cd $homepath

# Force library path to $homepath/lib.  Part of reducing our dependencies to
# none at all.
#
# The 'exec' overlays the shell so that there's no unnecessary shells haning
# about.
#
export GOGC=20

cmd="start"
if [ ! -z "$1" ] ; then
   cmd=$1
fi
PIDFILE=/var/tmp/link-share.pid

STDOUT=/var/log/code-ointment/link-share/link-share.stdout
STDERR=/var/log/code-ointment/link-share/link-share.stderr

if [ $cmd = "start" ]; then
    exec $homepath/bin/link-share > $STDOUT 2>$STDERR
elif [ $cmd = "stop" ]; then
    if [ ! -f $PIDFILE  ] ; then
        echo "Missing pid file, shutdown by hand please"
        exit 1
    fi
    PID=$(cat $PIDFILE)
    kill -TERM  $PID
fi
