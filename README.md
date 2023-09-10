Hydrofoil
======


### Introduction
Hydrofoil is a new linearizable replicated state machine (RSM) distributed algorithm that uses a combination of leader-based and leaderless protocols. This allows Hydrofoil to maintain the high throughput and reliability of leader-based RSMs, while reducing downtime and slowdowns when leaders fail by automatically switching to a leaderless algorithm.

### Details
Hydrofoil design is influenced by two other algorithms: Raft, a leader-based RSM that serves as the basis of many distributed databases in use today, and Ben-Or, a theoretical leaderless consensus algorithm that can operate even under asynchronous network conditions. Hydrofoil runs variations of both algorithms (which we denote as Raft+ and Ben-Or+ respectively) in parallel. Under normal network conditions, the Raft+ takes precedence, allowing for fast commits with high throughput even in georeplicated scenarios. When the leader is slow or offline, Ben-Or+ automatically takes over, enabling nearly uninterrupted performance during times when ordinary leader-based RSMs would need to trigger a re-election. The result is a linearizable RSM that is fault tolerant up to a minority of replica failures, with high throughput and low latency across a wide range of scenarios.

A full description of the protocol, and a largely complete proof can be found [here](Hydrofoil.pdf).

### Running the Code
Running Hydrofoil requires three components: a coordinator/master, any (odd) number of servers/replicas for replication, and a client to send messages to the RSM.

#### Build
Run `make` to build the master, replicas, and client using the given [Makefile](Makefile).

#### Running
Use [runAll.sh](runAll.sh) to run Hydrofoil. Use [killall.sh](killall.sh) to terminate the experiment. 

#### Tests
To run tests using a simulated connection, `cd` to `src/hydrofoil/` and run `go test -run Test`.

### Repository Details
This repo contains implementations of the following distributed protocols
* Hydrofoil
* Raft+
* Ben-Or+
* Copilot
* Latent Copilot
* EPaxos
* (Classic) Paxos
* Mencius
* Generalized Paxos

It is a fork of the [Copilot project](https://github.com/princeton-sns/copilot), which was developed by Khiem Ngo and Wyatt Lloyd. The implementations of EPaxos, MultiPaxos, Mencius, and Generalized Paxos were written by Iulian Moraru, David G. Andersen, and Michael Kaminsky respectively and come from the [EPaxos project](https://github.com/efficient/epaxos).

The [gobin-codegen](https://github.com/efficient/gobin-codegen) tool was used to generate marshaling and unmarshaling functions.


### About this Project
This work serves as my senior thesis for my undergraduate computer science degree at Princeton University. It was certainly a stressful but exciting experience!

I would like to thank Professor Wyatt Lloyd for being an amazing advisor throughout this entire process!