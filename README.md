# JDowser: Quick Start Guide

## Contents

* [JDowser Overview](#jdowser-overview)
* [Build JDowser](#build-jdowser)
* [JDowser usage](#jdowser-usage)
* [Sample JDowser run](#sample-jdowser-run)
* [Remote JDowser usage with Ansible](#use-jdowser-with-ansible)


## JDowser Overview

JDowser is a command-line tool that can locate all the Java installations on local and network drives.
JDowser recursively searches for Java installations starting from the root directory that is provided as a startup parameter.
This section describes the JDowser usage and shows how to manage the application remotely with Ansible.
The application is available for the Linux platform.


## Build JDowser

You can build JDowser from the source code that is freely available on GitHub.
The application is written in Golang, so you need to have a Golang package [installed](https://golang.org/doc/install).

To build JDowser:

1. Clone the JDowser project from the GitHub repository.
2. In your terminal run `make build-dist`:
   ```shell
   $ make build-dist
   ```
   This will generate the `jdowser` executable file in the `dist` folder.

You can also use the already built binaries from the **Releases** section.


## JDowser usage

To use JDowser, run the `jdowser` executable with a command and one or more optional parameters as shown below:

```shell
  jdowser [-json|-csv] [-skipfs fstype[,fstype..]] [-nojvmrun] [-root=<scanroot>] [-wait] start
  jdowser [-json|-csv] [-wait] status
  jdowser [-json|-csv] [-wait] report
  jdowser [-json|-csv] stop
```

The supported commands are the following:

* **start**: Starts scanning of the file system for Java installations. After the scan is complete, the application stops automatically.
* **status**: Displays the current application state. The possible states are *Running*, *Finished*, *Terminated*, *Error*, and *Unknown*.
* **report**: Displays the list of detected Java installations. If you run this command while the scanning is still in progress, you might get an incomplete list of Java installations detected so far.
* **stop**: Stops scanning of the file system.


The supported parameters are listed below. All the parameters are optional:

* **[-json|-csv]**: Sets the output format to JSON or CSV. By default, JDowser outputs text in a human-readable format.
* **[-skipfs fstype[,fstype..]]**: Defines file system types to skip.
If this parameter is not set, the default file systems to skip are `nfs,tmp,proc`.
If you specify this parameter, the default values are ignored.

* **[-nojvmrun]**: Instructs JDowser not to use `java -version` under the hood.

  By default, JDowser executes `java -version` to retrieve information about detected JVMs.
  Since some Java vendors may require a special license to run Java for commercial use, you can use the `-nojvmrun` parameter.
  With this parameter, JDowser uses alternative methods to analyze detected Java instances.
  These methods include scanning of JVM files (.jar, .so, etc.) and may produce less accurate results.

* **[-root=\<scanroot\>]**: Sets a root directory for scanning. The default path is `/`.
* **[-wait]**: Runs JDowser in foreground so the terminal waits until the scanning is complete.


## Sample JDowser run

Start scanning filesystem, this will run in background.

```shell
$ ./jdowser -root=/opt start
```

Or

```shell
$ ./jdowser start
```

Get scanning status Running/Finished.

```shell
$ ./jdowser status

host: host
state: Running
start_time: 2020-08-28 12:08:51 -0700 PDT
end_time: -1
args: -root=/opt
```

You can output the status to json format

```shell
$ ./jdowser -json status

{
  "host": "host",
  "state": "Finished",
  "start_time": 1598641731,
  "end_time": 1598641733,
  "args": [
    "-root=/opt"
  ]
}
```

And, finally you can get a report simple (as below) in json or csv format. 

```shell
$ ./jdowser report

host: host
libjvm: /opt/jvm/zulu-11-amd64/lib/server/libjvm.so
libjvm_hash: 1e4f56ecacb3458513beb17537197c2b
java_home: /opt/jvm/zulu-11-amd64
is_jdk: true
java_version: 11.0.7
java_runtime_name: Zulu11.39+15-CA
java_runtime_version: 11.0.7+10-LTS
java_runtime_vendor: Azul Systems, Inc.
java_vm_name: OpenJDK 64-Bit Server VM
java_vm_version: 11.0.7+10-LTS
java_vm_vendor: Azul Systems, Inc.
```

These commands will save data into specified files:
```shell
$ ./jdowser -json report > output_file.json
$ ./jdowser -csv report > output_file.csv
```


## Use JDowser with Ansible

The JDowser project directory contains the wrapper script `ansible-jdowser` that shows how to manage JDowser over the network using Ansible.
In this case, all authorization/authentication is delegated to Ansible.
For more information, see [Ansible documentation](https://docs.ansible.com/).

**NOTE**: The `ansible-jdowser` script has been verified to work with Ansible 2.9 and later.


### Install Ansible

To install Ansible, follow [Ansible Installation Guide](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html).


### Run JDowser with Ansible

To run JDowser with Ansible, run the `ansible-jdowser` script as shown in the example below:

```shell
ansible-jdowser [ansible args] start|stop|status|report [-csv|-json] [-wait]
```

The script startup options are the same as with direct JDowser invocation. However, some parameters are pre-defined in the script: `-nojvmrun`, `-root=/`
Appending `-wait` to the end of the command-line adds `-wait` to the JDowser arguments list.
You can also change the output format from the default JSON to CSV by passing the `-csv` parameter.


### Sample `ansible-jdowser` run

```shell
$ ./ansible-jdowser -i "host1,host2," start

[
  {
    "host": "host1",
    "state": "Running",
    "start_time": 1598670527,
    "end_time": -1,
    "args": [
      "-root=/",
      "-nojvmrun",
      "-json"
    ]
  },
  {
    "host": "host2",
    "state": "Running",
    "start_time": 1598670527,
    "end_time": -1,
    "args": [
      "-root=/",
      "-nojvmrun",
      "-json"
    ]
  }
]
```

<!-- ### Handle Ansible authentication

If an Ansible node requires user/password authentication, you can create a file `inventory.txt` with user credentials in the following format:
```shell
host_or_ip ansible_user=<user_name> ansible_password=<user_password>
host_or_ip ansible_user=<user_name> ansible_password=<user_password>
...
```

Then, you can pass `inventory.txt` to the script as shown below:
```shell
$ ./ansible-jdowser -i inventory.txt start
...
``` -->