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
- [x] Pass full QParts object to metrics
- Add disjointness check for Path Similarity
- Send acknowledgements over control conn informing about the performance of completion -> not yet used in the paper
- [x] Add path states: Unknown,Probing,Selected,Stale,Inactive
- Switch to one big sending queue instead of per datastream queue
- Use magic formula (or just Congestion Window Size) as part size for streams to fetch parts of data to be transferred
- Restructure Scheduler to PathSelector
- Implement all path selection methods defined in the paper
- [x] Add PathSelectionResponsibility to handshake, switch racelisten/dial accordingly.
- OnPathDown in OptimizedConn

## Bugs
- Sometimes concurrent map write call
- Sending back after receiving file was stuck

## Future Work
- Independent Path Selection for both directions