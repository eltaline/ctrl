Version: 1.1.8
========

Added in version 1.1.8:

- Change the time of tasks in queues

Version: 1.1.7
========

Incompatibilities:

- Building under solaris / darwin is no longer supported due to lack of pgid support

Fixed in version 1.1.7:

- Fixed working queue hanging
- Fixed hanging go routines
- Fixed process termination logic
- Fixed termination of child processes of executable commands

Version: 1.1.6
========

Fixed in version 1.1.6:

- May be was fix stuck working queue

Version: 1.1.5
========

Added in version 1.1.5:

- ```interr, intcnt``` fields for intercept errors and kill/repeat tasks on the fly
- ```vinterr, vintcnt``` fields for intercept errors and kill/repeat tasks on the fly in server configuration
- ```lookout``` boolean, enable or disable reaction on errors from stdout

Fixed in version 1.1.5:

- Clearing the working queue, in case of regular and after crash, server restart
- Improved extended error logging in app.log
- Minor bug fixes

Version: 1.1.4
========

Added in version 1.1.4:

- Return error status code from shell

Version: 1.1.3
========

Added in version 1.1.3:

- Field replace for /task

Version: 1.1.2
========

Fixed in version 1.1.2:

- Error mutex in /run

Version: 1.1.1
========

Added in version 1.1.1:

- Search by type

Version: 1.1.0
========

Added in version 1.1.0:

- Manual control of the parameters ```vthreads/vtimeout/vttltime/vinterval/vrepeaterr/vrepeatcnt``` through the POST method
- Setting the thread limit for a specific type of task
- Parsing errors and automatic task completion
- Updated documentation

Fixed in version 1.1.0:

- Minor bug fixes

Version: 1.0.0
========

- First Release
