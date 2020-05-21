#!/bin/bash

if [ -z `getent group ctrl` ]; then
	groupadd ctrl
fi

if [ -z `getent passwd ctrl` ]; then
	useradd ctrl -g ctrl -s /bin/sh
fi

install --mode=755 --owner=ctrl --group=ctrl --directory /var/lib/ctrl
install --mode=755 --owner=ctrl --group=ctrl --directory /var/log/ctrl

systemctl daemon-reload

#END