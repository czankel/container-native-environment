# Container Native Environment (CNE)

**Container Native Environment** (CNE) is a tool for building and managing
a virtual environment based on containers It provides users a reliable and
reproducible environment for development and applications, such as machine
learning or analytics.

Users can execute commands from the container transparently in the current
directory using their default settings and configurations. 

# Quick Start

The following steps demonstrate how to quickly create a container-native
environment, adding compiler and other build essentials to to that container
and compiling 'hello world'.

## Create a new CNE project

1. Create a directory for the project and change to it:
   `mkdir my-project`  
   `cd my-project`  
1. Initialize the container-native project. This will create the _cneproject_
   file in the current directory that describes the project. We use Ubuntu
   for the base image: 
   `cne init --image ubuntu`  
1. You can now run a command insize the container environment:  
   `cne exec cat /etc/os-release`  
   `cne exec ls`  

All commands executed with `cne exec` run inside the container with the
privileges of the current user and in the current directory. 

## Define the specific and reproducable environment


1. Create a new layer for managing the Apt packages: 
   `cne create layer -s apt`
1. Add the development packages:  
   `cne install apt build-essential`  
1. Compile 'hello world':  
   `echo -e '#include <stdio.h>\nint main(void) { printf("Hello World!\\n"); }\n' > test.c`  
   `cne exec -- gcc -o test test.c`  
1. The executbale can now be run inside the container environment or
   outside (if compatible):  
   `cne exec test`  
   `./test`  

## Define an alias to simplify the execution command

Having to always type `cne exec --` before the command can be simplified
with an alias to a single command. Using the alias defined below, you can
use:

`c ls -l` 

Instead of:

`cne exec -- ls -l` 

Add this alias to your .profile or .bashrc:

`alias c='cne exec -- '`

The trailing space is an indication for bash to also check the command word
following this alias for alias expansion. For example, `c ll` would expand
the `ll` alias (if defined). 

This can also be used to execute commands inside the container environment
with root privileges by defining the alias:

`alias sudo='sudo '`

Using `sudo c id` will then display root as the current user.


## Clean

The container environment will automatically be re-created if it has been
destroyed. If you don't need the containers anymore or if there are any other
issues, try to delete all resources associated witht the project:

`cne clean project`
