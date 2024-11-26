# Main TODOS

- [x] Proper Logging
- [x] Proper error handling
- [x] Proper number of QUICDataplaneStreams
- [x] Hide quic go not a udp conn message
- [x] Return proper numbers of Read/Write from API
- [x] Communicate paths from conn to QUICDataplaneStream for reverse scheduling
- [x] Implement Deadline functionalities
- Stream Types
- Priorities
- [x] Crypto
- MTU Discovery and handling
- [x] Send loop
- Change path on the fly for optimizedConn
- Add metrics for both directions -> path from optimizedConn -> reverse?
- Complete QuicConnTracer
- Add UMCC
- [x] Wait for completion of sequence / sequence Ack? -> not mandatorily
- Ignore timeouts for dataplane streams, send keepalives
- Handle timeouts in racedial/listendial properly
- Handle pathdown / errors when sending -> trigger event

## Bugs