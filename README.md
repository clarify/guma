# Guma (deprecated)

**Disclaimer: The code as it stands, is released as an inspiration to others, but does not offer a high-level interface and probably contain some bugs.**

## History

In [Searis](http://searis.no), we have the experience from having developed a OPC UA library in Go already back in 2014, although for a specific purpose, so GUMA was actually our second attempt where we started from scratch, and moved from mostly hand-written code to mostly generated code.

GUMA was written to be a general purpose open source [OPC UA](https://opcfoundation.org/about/opc-technologies/opc-ua/) client library for the [Go](https://golang.org) programming language. However, due to priorities and limited resources we never made it so far, and the project has been on hold since Nov 22, 2017, shrotly after the work had started.

Recently we have dicovered that there is [another](https://github.com/gopcua/opcua) open source initative for OPC UA in Go that has gained traction, and we have chosen to finally deprecate this project. However, we also want to share what we have learned, and have chosen to open soure the code as an inspiration to others under an MIT License. It's also possible to fork or adopt this project if the code base is of particular interest to anyone.

![Image of the Guma](/misc/img/guma.png)

## Repository structure

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

## Status

- [x] Handshake and opening of Secure Channel.
- [ ] Closing a secure channel (easy, but not implemented).
- [x] Access to all OPC UA Service calls, such as Read, Browse and Subscribe.
- [x] SecureChannel made safe for concurrent access (necesary for e.g. Subscribe).
- [ ] Secure channel Message signing and encryption.
- [ ] Stateless HTTPS / HTTP.
- [ ] Reconnect TCP Socket on errors.
- [ ] Re-new Secure Channels at 75% of revised lifetine.

## Development

To re-generate code, we rely on [Go task](https://taskfile.dev/#/):

    task generate

See all tasks with `task -l`.

Known issues: patch to schema XML files no longer applies (OPC UA 1.03):

```sh
$ task download-schema
...
Hunk #175 succeeded at 2361 (offset 30 lines).
Hunk #176 FAILED at 2375.
1 out of 176 hunks FAILED -- saving rejects to file schemas/1.03/Opc.Ua.Types.bsd.xml.rej
$ cat schemas/1.03/Opc.Ua.Types.bsd.xml.rej                                                                              :(
***************
*** 2378
- </opc:TypeDictionary>--- 2375 -----
+ </opc:TypeDictionary>
```

Run unit-tests:

    dep ensure
    go test -v ./...
