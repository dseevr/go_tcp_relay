# Go TCP Relay

This was written in early October, 2014.  I had a MT4 EA (MetaTrader4 Expert Advisor) writing tick data from multiple charts out to sockets.  I wrote this relay so it could take N writers and stream a copy of everything they sent to the relay out to an arbitary number of consumers.  One consumer would log to disk, another would be a trading algorithm server, etc.

I used a newline-delimited text-based messaging protocol, so the keepalives to the consumers were just a "PING\n".

## License

BSD
