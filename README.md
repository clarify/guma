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
- stack
  - A low level client for talking to OPC UA servers
- stack/uatype:
  - A package exporting a set of Go types that has mostly been generated from source.
- stack/transport:
  - A parent package for OPC UA Secure Channel implementations.
- stack/ecoding/binary:
  - A package similar to `encoding/json` in the standard library, that allows encoding of structs into an OPC UA binary representation.


## Features

- [x] Handshake and opening of Secure Channel.
- [ ] Closing a secure channel (easy, but not implemented).
- [x] Access to all OPC UA Service calls, such as Read, Browse and Subscribe.
- [x] SecureChannel made safe for concurrent access (necesary for e.g. Subscribe).
- [ ] Secure channel Message signing and encryption.
- [ ] Stateless HTTPS / HTTP.
- [ ] Reconnect TCP Socket on errors.
- [ ] Re-new Secure Channels at 75% of revised lifetine.

## Long term goals

- [ ] Certify an application built with this library with the OPC Fondation.


## Development

To re-generate code, we rely on go-task (broken patches needs fixing!):

```bash
$ go get -u -v github.com/go-task/task/cmd/task
$ task generate
```

Run unit-tests:

```bash
$ dep ensure
$ go test -v ./...
```


![Image of the Guma](/misc/img/guma.png)
