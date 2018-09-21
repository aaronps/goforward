# Go Forwarder

TCP and UDP port forwarding.

### Usage

```sh
goforward <tcp|udp> <listen_address> <target_address1> [...<target_addressN>]
```

Target addresses will be selected using round-robin algorithm.

### Examples

```sh
# listen in port 8080, forward to 1.2.3.4:80
goforward tcp :8080 1.2.3.4:80

# listen in port 8080, forward connections alternating between two servers
goforward tcp :8080 1.2.3.4:80 5.6.7.8:80

# forward udp packets from port 20000 to alternating servers
goforward udp :20000 1.2.3.4:20000 5.6.7.8:20000

# listen only on localhost:10022 and forward to 1.2.3.4:22
goforward tcp 127.0.0.1:10022 1.2.3.4:22
```
