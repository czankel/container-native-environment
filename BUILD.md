# Build Instructions

CNE must run with root privileges to communicate with the containerd daemon.
Makefile, therefore, uses sudo in the install rule to set the suid flag and
change the owner to root.

To build `cne`, use:

`make`

To install `cne`, use:

`make install`

This will install `cne` under `/usr/local/bin`. If you want to install it to
a different location, you can provide a different path with `DESTDIR=<path>`:

`make install DESTDIR=/usr/bin`

Use this command to run some unittests:

`make test`
