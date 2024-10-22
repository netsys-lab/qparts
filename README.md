# Q-PARTS: QUIC-based Path-Aware Reliable Transport over SCION

## Problem to solve
- SCION provides path-aware networking (PAN) and a potentially large number of heterogenous paths to a given destination
- Existing Multipath transport protocols, that define a path as a 4-tuple, do not support PAN via SCION
- MPTCP/MPQUIC work well with a small number of homogenous paths, but are not optimized for larger scenarios
- Consequently: SCION is missing a native, tailored multipath transport protocol

## Core ideas
- QUIC provides a great transport protocol to transfer data over a single path
- QUIC is not designed to switch the underlying path frequently or distribute traffic over multiple underlying paths
- Pinning a QUIC connection to a SCION path provides a great foundation to achieve multipath transport over SCION
- Assumption: A data sequence sent over a QUIC stream shares similar properties, e.g. being latency sensitive, targeting for high throughput, ...
- Providing a way for application to communicate these properties to the transport protocol offers huge optimisaion potential

## Architecture
Q-PARTS combines the following ideas and concepts:
- Q-PARTS offers a similar API than QUIC (quic-go): Listener, Connection, Stream, etc called QPartsListener, QPartsConn, QPartsStream
- Q-PARTS runs multiple QUIC Connection/Stream pairs in the background, each pair over a dedicated SCION path
- QPartsStream works similar to QUIC Streams, except that they are distributed over a single, or multiple QUIC streams in the background and consequently over multiple SCION paths
- A data sequence to be sent over a QPartsStream is split into one or more "parts", each part is transferred over one QUIC stream
- QUICs monitoring capabilities are aggregated to detect shared bottlenecks (as explained in UMCC), and Q-PARTS ensuers that only a single QUIC stream is sent through a shared bottleneck to be fair to others
- Q-PARTS offers to use PILA (Pervasive Internet-Wide Low-Latency Authentication) to obtain host certificates from SCION's control plane to authenticate one or both sides of the connection

## Overview sketch
![qparts](https://github.com/user-attachments/assets/104954b5-97e3-480c-8443-7504e6a8865b)

Description:
- Sender is transferring to receiver
- Two QPartsStreams are open with different properties (one for low latency, one for high throughput)
- Q-Parts Scheduler sends data sequence of Stream1 over the shortest, direct SCION path (into one "part")
- Q-Parts Scheduler splits data sequence of Stream2 over two longer paths (into 2 "parts")
- Q-Parts re-assembles the two "parts" of Stream2 at the receiving side and transfers the sequence in the right order to the receiving application

## API
### Replace QUIC with Q-PARTS
quic-co receiving side :
```go
// Setup UDP Conn
conn, err := net.ListenUDP("udp", udpAddr)

// QUIC Listener
listener, err := quic.Listen(conn, &tls.Config{InsecureSkipVerify: true, Certificates: MustGenerateSelfSignedCert(), NextProtos: []string{"qparts"}}, &quic.Config{})

// QUIC Conn
sess, err := listener.Accept(context.Background())

// QUIC Stream
stream, err := sess.AcceptStream(context.Background())

// Read from Stream
p := make([]byte, 1024)
n, err := stream.Read(p)
```

Q-PARTS receiving side:
```go
// UDP conn is created automatically

// Q-PARTS Listener
listener, err := qparts.Listen(local, &tls.Config{InsecureSkipVerify: true, Certificates: MustGenerateSelfSignedCert(), NextProtos: []string{"qparts"}}, &quic.Config{})

// Q-PARTS Conn
sess, err := listener.Accept(context.Background())

// Q-PARTS Stream
stream, err := sess.AcceptStream(context.Background())

// Read from Stream
p := make([]byte, 1024)
n, err := stream.Read(p)
```

quic-co sending side :
```go
// Setup UDP Conn
conn, err := net.ListenUDP("udp", udpAddr)

// Open Conn
session, err := quic.Dial(context.Background(), conn, remoteAddr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"qparts"}}, &quic.Config{})

// Open Stream
stream, err := session.OpenStreamSync(context.Background())

// Write to Stream
p := make([]byte, 1024)
n, err := stream.Write(p)
```

Q-PARTS sending side:
```go
// UDP Conn created automatically

// Q-PARTS Conn
session, err := qparts.Dial(context.Background(), localAddr, remoteAddr, &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"qparts"}}, &quic.Config{})

// Q-PARTS Stream
stream, err := session.OpenStreamSync(context.Background())

// Write to Stream
p := make([]byte, 1024)
n, err := stream.Write(p)
```


### Extended API
tbd..

## Notes
- Ideally, Q-PARTS would be integrated into MPQUIC, however it seems the current MPQUIC implementation is a) not up to date with the recent quic-go version and b) is not implementing all points mentioned in the RFC
