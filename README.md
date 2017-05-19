# Guma

Guma will be our new open source [OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) library for the [Go](https://golang.org) programming language. In [Searis](http://searis.no), we have the experience from having developed a OPC UA library in Go before in cooperation with our partners. When we now do a new and awesome open source library, we want to do it properly rather than fast.

Initially we are aiming to first create the building blocs that are needed for creating an OPC UA Client, rather than to offer any specific support for creating servers.

Watch out for Guma!


## Content

At the moment, this repo includes the following:

- generate/scripts:
  - A script to download the OPC UA XML and CSV schemas.
- generate/cmd:
  - Commands used to generate code structures from XML and CSV.
- uatype:
  - A package exporting a set of Go types that has mostly been generated from source.

To re-generate code, we rely on go-task:

```bash
$ go get -u -v github.com/go-task/task/cmd/task
$ task generate
```

Run unit-tests:

```bash
$ glide install
$ go test -v
```

Dependencies is managed via [Glide](http://glide.sh) until Go's new official dependency tool is ready. Run `glide init` or `glide cw` to use Glide on your own projects.

## Features:

- [x] Auto generate a large selection of structures and types with struct tags.
- [ ] Provide an encoder for encoding all supported data structures to the OPC UA binary format.
- [ ] Provide a decoder for decoding the OPC UA binary format into Go structs.
- [ ] Support code for creating OPC UA Client applications:
  - [ ] Basic message handling logic.
  - [ ] Basic client connection capabilities.
  - [ ] User authentication support.
  - [ ] Read and Browse capabilities.
  - [ ] Subscription support.
- [ ] Support for creating OPC UA server applications?

![Image of the Guma](/misc/img/guma.png)
