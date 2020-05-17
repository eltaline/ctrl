cTRL Parameters:
========

Section [command line]
------------

- -config string
        --config=/etc/ctrl/ctrl.conf (default is "/etc/ctrl/ctrl.conf")
- -debug 
        --debug - debug mode
- -help 
        --help - displays help
- -version 
        --version - print version

Section [global]
------------

bindaddr
- **Description**: This is the primary address and TCP port. The value is ": 9699" for all addresses.
- **Default:** "127.0.0.1:9699"
- **Type:** string
- **Section:** [global]

bindaddrssl
- **Description:** This is the primary SSL address and TCP port. The value is ": 9699" for all addresses.
- **Default:** "127.0.0.1:9799"
- **Type:** string
- **Section:** [global]

onlyssl
- **Description:** This globally disables standart HTTP address and port for all virtual hosts. It is not configured for a virtual host.
- **Default:** false
- **Values:** true or false
- **Type:** bool
- **Section:** [global]

readtimeout
- **Description:** This sets the timeout for the maximum data transfer time from the client to the server. It should be increased to transfer large files. If the cTRL server is installed behind Nginx or HAProxy, this timeout can be disabled by setting it to 0 (no timeout). It is not configured for a virtual host.
- **Default:** 60
- **Values:** 0-86400
- **Type:** int
- **Section:** [global]

readheadertimeout
- **Description:** This sets the timeout for the maximum time for receiving headers from the client. If the cTRL server is installed behind Nginx or HAProxy, this timeout can be disabled by setting it to 0. If the parameter = 0, the readtimeout parameter is taken. If the readtimeout = 0, the readheadertimeout timeout is not used (no timeout). It is not configured for a virtual host.
- **Default:** 5
- **Values:** 0-86400
- **Type:** int
- **Section:** [global]

writetimeout
- **Description:** This sets the timeout for the maximum data transfer time to the client. The transfer of large files should be significantly increased. If the cTRL server is installed behind Nginx or HAProxy, this timeout can be disabled by setting it to 0 (no timeout). It is not configured for a virtual host.
- **Default:** 60
- **Values:** 0-86400
- **Type:** int
- **Section:** [global]

idletimeout
- **Description:** This sets the timeout for the maximum lifetime of keep alive connections. If the parameter = 0, the readtimeout parameter value is taken. If the readtimeout = 0, the idletimeout timeout is not used (no timeout). It is not configured for a virtual host.
- **Default:** 60
- **Values:** 0-86400
- **Type:** int
- **Section:** [global]

keepalive
- **Description:** This globally enables or disables keep alive. It is not configured for a virtual host.
- **Default:** false
- **Values:** true or false
- **Type:** bool
- **Section:** [global]

realheader
- **Description:** This is the real IP address header from the reverse proxy. It is not configured for a virtual host.
- **Default:** "X-Real-IP"
- **Type:** string
- **Section:** [global]
       
charset
- **Description:** This is the encoding used for the entire server. It is not configured for a virtual host.
- **Default:** "UTF-8"
- **Type:** string
- **Section:** [global]

debugmode
- **Description:** This globally enables debug mode. It is not configured for a virtual host.
- **Default:** false
- **Values:** true or false
- **Type:** bool
- **Section:** [global]

gcpercent
- **Description:** This globally sets the garbage collection parameter (percent). -1 disables the garbage collector. It is not configured for a virtual host.
- **Default:** 25
- **Values:** -1-100
- **Type:** int
- **Section:** [global]

dbdir
- **Description:** Directory for tasks database. You can safely remove data and reinitialize again, in this case compeleted task too deleted.
- **Default:** "/var/lib/ctrl/db"
- **Type:** string
- **Section:** [global]

schtime = 5
- **Description:** This is the compaction/defragmentation delay timeout for the updated Bolt archives (days).
- **Default:** 5
- **Values:** 1-3600
- **Type:** int
- **Section:** [global]

pidfile
- **Description:** This is the PID file path.
- **Default:** "/run/ctrl/ctrl.pid"
- **Type:** string
- **Section:** [global]

logdir
- **Description:** This is the path to the log directory.
- **Default:** "/var/log/ctrl"
- **Type:** string
- **Section:** [global]

logmode
- **Description:** This sets the permissions for the log files (mask).
- **Default:** 0640
- **Values:** 0600-0666
- **Type:** uint32
- **Section:** [global]

Section [server] and subsections [server.name]
------------

[server.name]
- **Description:** This is the primary internal identifier of the virtual host. After "." any name can be used. This is not a domain name, only an internal identifier.
- **Type:** string
- **Section:** [server]

host
- **Description:** This is the virtual host name. The value * or _ is not supported. To convert multiple virtual hosts to one virtual host in a cTRL server, use Nginx or HAProxy, or any other reverse proxy server with a hard-set "proxy_set_header Host hostname;" (using Nginx as an example) where hostname = host in the cTRL server virtual host.
- **Default:** Required
- **Type:** string
- **Section:** [server.name]

sslcrt
- **Description:** This is the path to the file with SSL certificate of virtual host.
- **Default:** ""
- **Type:** string
- **Section:** [server.name]

sslkey
- **Description:** This is the path to the file with SSL key of virtual host.
- **Default:** ""
- **Type:** string
- **Section:** [server.name]

ussallow
- **Description:** This is the path to the file of login:pass(sha512) pairs for the GET/POST methods for the virtual host.
- **Default:** ""
- **Type:** string
- **Section:** [server.name]

ipsallow
- **Description:** This is the path to the file of allowed IP addresses for the GET/POST methods for the virtual host.
- **Default:** ""
- **Type:** string
- **Section:** [server.name]

shell
- **Description:** Sets the shell for the virtual host.
- **Default:** "/bin/bash"
- **Type:** string
- **Section:** [server.name]

rthreads
- **Description:** Maximum number of concurrently running parallel tasks in real time.
- **Default:** 8
- **Values:** 1-4096
- **Type:** int
- **Section:** [server.name]

vthreads
- **Description:** Maximum number of concurrently running parallel tasks in queue mode.
- **Default:** 8
- **Values:** 1-4096
- **Type:** int
- **Section:** [server.name]

vtimeout
- **Description:** This sets default timeout for running tasks, if timeout is already not set in task.
- **Default:** 28800
- **Values:** 1-2592000
- **Type:** uint32
- **Section:** [server.name]

vttltime
- **Description:** This sets default time to live for completed tasks in database.
- **Default:** 86400
- **Values:** 1-2592000
- **Type:** uint32
- **Section:** [server.name]
