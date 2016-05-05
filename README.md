cephbroker
===========

This is a service broker written in Go Language for Cloud Foundry. This service broker supports creating Distributed File Systems provided by a  Ceph cluster.

Getting Started
===============

Get Latest Executable: cephbroker
----------------------------------------

Assuming you have a valid [Golang 1.4.2](https://golang.org/dl/) or [later](https://golang.org/dl/) installed for your system, you can quickly build and get the latest `go_service_broker` executable by running the following `go` command:

```
$ go get github.com/cloudfoundry-incubator/cephbroker
```

This will build and place the `cephbroker` executable built for your operating system in your `$GOPATH/bin` directory.


Building From Source
--------------------

Clone this repo and build it. Using the following commands on a Linux or Mac OS X system:

```
$ mkdir -p cephbroker/src/github.com/cloudfoundry-incubator
$ export GOPATH=$(pwd)/cephbroker:$GOPATH
$ cd cephbroker/src/github.com/cloudfoundry-incubator
$ git clone https://github.com/cloudfoundry-incubator/cephbroker.git
$ cd cephbroker
$ ./bin/build
```

NOTE2: if you get any dependency errors, then use `go get path/to/dependency` to get it, e.g., `go get github.com/onsi/ginkgo` and `go get github.com/onsi/gomega`

The executable output should now be located in: `out/cephbroker`. Place it wherever you want, e.g., `/usr/local/bin` on Linux or Mac OS X.

Configuring for a specific Ceph Cluster
=======================================

TBD

Running Broker
==============

The broker can be ran in one of two modes: locally or as an app in a CF environment.

Locally
-------

Run the executable to start the service broker which will listening on port `8001` by default.

```
$ out/cephbroker
```

Using Broker
============

TBD

License
=======
This is under [Apache 2.0 OSS license](https://github.com/cloudfoundry-samples/go_service_broker/LICENSE).
