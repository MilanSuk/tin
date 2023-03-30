# Tin
Tin is an **experimental** cryptocurrency. The goal is to push some blockchain's properties to another level, but keep decentralization large as possible.

Tin source code is written from scratch. It's not based on any other blockchain.



## Research
High transaction throughput:
- Bitcoin can process 3-7 txns/sec. It has a 1MB block every ~10 minutes
- Tin can do 30-100 txns/sec, with the same 1MB every 10 minutes as Bitcoin
- Higher throughput => lower fees for senders or higher income for miners

Light-client(work in progress):
- Download data directly from node(no "trusted" centralized server needed)
- Access from the browser



## Performance
user : 51 sec to generate 40K txns
miner: 12 sec to create a block with 10K txns(~0.15MB). A full 1MB block can hold ~60K txns
miner: 5 sec to check block(10K txns)

1MB block(60K txns) mined every 10 minutes = 100 txns/sec



## Compile & Run
Libraries
<pre><code>go get github.com/gorilla/websocket
go get github.com/mattn/go-sqlite3
go get github.com/herumi/bls-eth-go-binary
</code></pre>

Tin
<pre><code>git clone https://github.com/milansuk/tin
cd tin
go build
./tin
</code></pre>



## Libraries
- SQLite for ledger
- WebSocket for client-server
- BLS for crypto



## Author
Milan Suk

Email: milan@skyalt.com

Twitter: https://twitter.com/milansuk/

*Feel free to follow or contact me with any idea, question or problem.*



## Contributing
Your feedback and code are welcome!

For bug reports or questions, please use GitHub's issues.

Tin is licensed under **Apache v2.0** license. This repository includes 100% of the code.