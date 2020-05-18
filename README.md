Документация на русском: https://github.com/eltaline/ctrl/blob/master/README-RUS.md

cTRL is a server written in Go language that uses a <a href=https://github.com/eltaline/nutsdb>modified</a> version of the NutsDB database, as a backend for a continuous queue of tasks and saving the result of executing commands from given tasks in command interpreters like /bin/bash on servers where this service will be used. Using cTRL, you can receive tasks via the HTTP protocol with commands for executing them on the server and limit the number of simultaneously executed tasks.

Current stable version: 1.0.0
========

- <a href=/CHANGELOG.md>Changelog</a>

Features
========

- Multi threading
- Supports HTTPS and Auth/IP authorization
- Supported HTTP methods: GET, POST
- Parallel execution of tasks in realtime
- Receiving tasks in the queue for deferred parallel execution of commands
- Limiting the maximum number of threads per virtual host
- Supports continuous task execution queue
- Automatic simultaneous execution of only one identical task
- Supported interpreters: /bin/bash, /bin/sh
- Supported formats: JSON

Requirements
========

- Operating Systems: Linux, BSD, Solaris, OSX
- Architectures: amd64, arm64, ppc64 and mips64, with only amd64 tested

Real application
========

We use this server to perform delayed tasks for parallel video conversion using ffmpeg with limiting processes on several video servers with Nvidia GPU, but the server can execute any commands through the shell interpreters. This is a first release.

Documentation
========

Installation
--------

Install packages or binaries
--------

- <a href=https://github.com/eltaline/ctrl/releases>Download</a>

```
systemctl enable ctrl && systemctl start ctrl
```

Configuring and using cTRL server
--------

Default credentials: ```admin:swordfish```

Password generation: ```echo -n "mypassword" | sha512sum```

In most cases it is enough to use the default configuration file. A full description of all product parameters is available here: <a href="/OPTIONS.md">Options</a>

This guide uses UUIDs. But the client can set task identifiers in any format.

General methods
--------

Run a task or task list in realtime with pending execution

```bash
curl -X POST -H "Auth: login:pass" -H "Content-Type: application/json" -d @task.json http://localhost/run
```

Queuing a task or task list

```bash
curl -X POST -H "Auth: login:pass" -H "Content-Type: application/json" -d @task.json http://localhost/task
```

Getting a task from the received queue

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=received"
```

Getting all tasks from the received queue

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=received"
```

Getting a task from the working queue

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=working"
```

Getting all tasks from the working queue

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=working"
```

Getting a task from the list of completed tasks

```bash
curl -H "Auth: login:pass" "http://localhost/show?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=completed"
```

Getting all tasks from the list of completed tasks

```bash
curl -H "Auth: login:pass" "http://localhost/show?queue=completed"
```

Deleting a task from the received queue

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=received"
```

Deleting all tasks from the received queue

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=received"
```

Deleting a task from the working queue

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=working"
```

Deleting all tasks from the working queue

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=working"
```

Deleting a task from the list of completed tasks

```bash
curl -H "Auth: login:pass" "http://localhost/del?key=777a0d24-289e-4615-a439-0bd4efab6103&queue=completed"
```

Deleting all tasks from the list of completed tasks

```bash
curl -H "Auth: login:pass" "http://localhost/del?queue=completed"
```

Input format
--------

Description of fields
--------

- key - an arbitrary unique identifier
- type - arbitrary type of task
- path - path to change the directory before running the command
- lock - arbitrary lock label
- command - command
- timeout - timeout

Examples of setting tasks
--------

Example for one task:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15}
]
```

Example for a task list:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","timeout":15},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","timeout":15}
]
```

Output format
--------

Description of fields
--------

- key - an arbitrary unique identifier
- time - task receive time
- type - arbitrary type of task
- path - path to change the directory before running the command
- lock - arbitrary lock label
- command - command
- timeout - timeout
- stdcode - not used at the moment
- stdout - standard output
- errcode - error code
- stderr - standard error output
- runtime - runtime in milliseconds
- delcode - error code when deleting a task
- delerr - error message when deleting a task

Output example
--------

Output of completed tasks:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","time":1589737139,"type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15,"stdcode":0,"stdout":"hello\n","errcode":0,"stderr":"","runtime":10010.669069},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","time":1589737139,"type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","timeout":15,"stdcode":0,"stdout":"great\n","errcode":124,"stderr":"signal: killed","runtime":15006.034832},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","time":1589737139,"type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","timeout":15,"stdcode":0,"stdout":"world\n","errcode":0,"stderr":"","runtime":15019.839685}
]
```

Output of deleted tasks:

```json
[
{"key":"777a0d24-289e-4615-a439-0bd4efab6103","time":1589737139,"type":"mytype","path":"/","lock":"mylock1","command":"echo \"hello\" && logger \"hello\" && sleep 5","timeout":15,"stdcode":0,"stdout":"hello\n","errcode":0,"stderr":"","runtime":10010.669069,"delcode":0,"delerr":""},
{"key":"4964deca-46ff-413f-8a92-e5baefd328e7","time":1589737139,"type":"mytype","path":"/","lock":"mylock2","command":"echo \"great\" && logger \"great\" && sleep 30","timeout":15,"stdcode":0,"stdout":"great\n","errcode":124,"stderr":"signal: killed","runtime":15006.034832,"delcode":0,"delerr":""},
{"key":"3fdf744d-36f1-499d-bd39-90a004ee39f6","time":1589737139,"type":"mytype","path":"/","lock":"mylock3","command":"echo \"world\" && logger \"world\" && sleep 15","timeout":15,"stdcode":0,"stdout":"world\n","errcode":0,"stderr":"","runtime":15019.839685,"delcode":0,"delerr":""}
]
```

Notes and Q&A
--------

- The limitation of simultaneously running tasks from queue to each virtual host is regulated by the ```vthreads``` parameter in the server configuration file
- The limitation of simultaneously running tasks in realtime to each virtual host is regulated by the ```rthreads``` parameter in the server configuration file
- The key field, if this identifier is the same for two or more different tasks, in this case, when outputting information from the queue, you will receive information on this identifier for several tasks at once, this can be useful for grouping tasks, but they will be from the waiting queue run randomly
- The type and lock fields, if they are assigned to two or more different tasks, are absolutely identical, in which case the server will perform these tasks from the wait queue in an arbitrary order, but only in turn
- The type and lock fields set in real time do not matter, but they are required, all tasks will be performed in parallel mode whenever possible.
- To sequentially execute a list of specific commands related to each other through a waiting queue, install these commands in one task, separated by && or write a shell script
- Tasks performed in real time are executed parallel.

Error Codes
--------

errcode
--------

- 0 (no error)
- 1 (any error)
- 124 (timeout)
- 255 (failed to start command)
- shell`s exit statuses available in stderr field

delcode
--------

- 0 (no error)
- 1 (any error)

Todo
========

- Any suggestions

Parameters
========

A full description of all product parameters is available here: <a href="/OPTIONS.md">Options</a>

HTTP Core
========

Uses <a href=https://github.com/kataras/iris>Iris</a> as server http core

Guarantees
========

No warranty is provided for this software. Please test first

Donations
========

<a href="https://www.paypal.me/xwzd"><img src="/images/paypal.png"><a/>

Contacts
========

- E-mail: dev@wzd.dev
- cTRL website: https://wzd.dev
- Company website: <a href="https://elta.ee">Eltaline</a>

```
Copyright © 2020 Andrey Kuvshinov. Contacts: <syslinux@protonmail.com>
Copyright © 2020 Eltaline OU. Contacts: <eltaline.ou@gmail.com>
All rights reserved.
```
