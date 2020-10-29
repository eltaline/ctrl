#!/bin/bash

if [ -z `getent group ctrl` ]; then
	groupadd ctrl
fi

if [ -z `getent passwd ctrl` ]; then
	useradd ctrl -g ctrl -s /bin/sh
fi

if [ ! -d /var/lib/ctrl ] ; then
	install --mode=755 --owner=ctrl --group=ctrl --directory /var/lib/ctrl
fi

if [ ! -d /var/log/ctrl ] ; then
	install --mode=755 --owner=ctrl --group=ctrl --directory /var/log/ctrl
fi

systemctl daemon-reload

#END