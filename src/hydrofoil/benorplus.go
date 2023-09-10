package hydrofoil

import (
	"log"
	"math/rand"
	"sort"
	"time"
)

/************************************** Ben Or Braodcast **********************************************/
func (r *Replica) startBenOrPlus() {
	if r.leaderState.isLeader {
		log.Fatal("Should not be starting ben or plus if I am leader")
	}

	// dlog.Println("Replica", r.Id, "hmmm3", "log is", logToString(r.log))
	r.startNewBenOrPlusIteration(0, make([]Entry, 0), make([]bool, r.N))
}

func (r *Replica) startNewBenOrPlusIteration(iteration int, benOrBroadcastMsg []Entry, heardServerFromBroadcast []bool) {
	var broadcastEntry Entry
	benOrIndex := r.commitIndex + 1
	// biasedCoin := false
	if benOrIndex < len(r.log) {
		// dlog.Println(r.Id, "Ben Or Plus: Broadcasting entry", benOrIndex, "from log")
		broadcastEntry = r.log[benOrIndex]
		broadcastEntry.Index = int32(r.commitIndex)+1
		// biasedCoin = true
	} else if !r.pq.isEmpty() {
		// dlog.Println(r.Id, "Ben Or Plus: Broadcasting entry", benOrIndex, "from pq, current commit indx:", r.commitIndex)
		broadcastEntry = r.pq.peek()
		broadcastEntry.Index = int32(r.commitIndex)+1
	} else if iteration == 0 {
		if r.leaderState.isLeader {
			log.Fatal("Should not be leader if log and pq are empty")
		}

		// fmt.Println("Replica", r.Id, "resetting ben or plus wait time")

		r.sendGetCommittedData()
		// nothing to run ben-or on right now, wait until timeout again
		timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
		setTimer(r.benOrStartTimer, time.Duration(timeout) * time.Millisecond)

		clearTimer(r.benOrResendTimer)
		return
	} else {
		// dlog.Println("Replica log", r.Id, "is", logToString(r.log), "pq is", logToString(r.pq.extractList()))
		log.Fatal("Ben Or Plus: No entry to broadcast")
	}

	heardServerFromBroadcast[r.Id] = true

	r.benOrState = BenOrState{
		benOrRunning: true,
	
		benOrIteration: iteration,
		benOrPhase: 0,
		benOrStage: Broadcasting,
	
		benOrBroadcastEntry: broadcastEntry,
		benOrBroadcastMsgs: append(benOrBroadcastMsg, broadcastEntry),
		heardServerFromBroadcast: heardServerFromBroadcast,

		haveMajEntry: false,
		benOrMajEntry: emptyEntry,

		benOrVote: VoteUninitialized,
		benOrConsensusMsgs: make([]uint8, 0),
		heardServerFromConsensus: make([]bool, r.N),

		prevPhaseFinalValue: VoteUninitialized,

		biasedCoin: false,
	}
	// fmt.Println("Replica", r.Id, "type1", time.Now().UnixMicro() - startTime)
	r.startBenOrBroadcast()
}

func (r *Replica) startBenOrBroadcast() {
	// startTime := time.Now().UnixMicro()
	entries := make([]Entry, 0)
	if r.commitIndex + 1 < len(r.log) {
		entries = r.log[r.commitIndex + 1:]
	}

	// dlog.Println("Replica", r.Id, "starting Ben Or Broadcast", r.benOrState.benOrIteration, r.benOrState.benOrBroadcastEntry.Data.OpId, r.benOrState.heardServerFromBroadcast, "with log", len(r.log), "pq entries: ", logToString(r.pq.extractList()))
	args := &BenOrBroadcast{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
		Iteration: int32(r.benOrState.benOrIteration), BroadcastEntry: r.benOrState.benOrBroadcastEntry,
		StartIndex: int32(r.commitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
	}

	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout) * time.Millisecond)

	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
		}
	}
}

func (r *Replica) updateLogIfPossible(rpc ReplyMsg) bool {
	if r.isLogMoreUpToDate(rpc) == LessUpToDate {
		if r.commitIndex + 1 >= int(rpc.GetStartIndex()) {
			r.updateLogFromRPC(rpc)
			return true
		} else {
			// Otherwise, we remain out of date. If this is the case, then we wait until we're also running BenOrPlus.
			// When we send a BenOrPlus message and we're out of date, we'll get back data allowing us to update our log.
			return false
		}
	} else {
		for _, v := range(rpc.GetEntries()) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}
	
		for _, v := range(rpc.GetPQEntries()) {
			if !r.seenBefore(v) { r.pq.push(v) }
		}

		if r.leaderState.isLeader {
			for !r.pq.isEmpty() {
				entry := r.pq.pop()
				entry.Term = int32(r.term)
				entry.Index = int32(len(r.log))

				if !r.inLog.contains(entry) {
					r.log = append(r.log, entry)
					r.inLog.add(entry)
				}
			}
			r.leaderState.repNextIndex[r.Id] = len(r.log)
			r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
		}
		return true
	}
}

func (r *Replica) handleBenOrBroadcast(rpc BenOrBroadcastMsg) {
	r.handleIncomingTerm(rpc)

	if !r.updateLogIfPossible(rpc) {
		// dlog.Println("Replica", r.Id, "is out of date, waiting for BenOrPlus to update log")
		r.sendOneGetCommittedData(int(rpc.GetSenderId()))
		return
	}

	if rpc.GetBenOrMsgValid() == False {
		// dlog.Println("Replica", r.Id, "received invalid BenOrBroadcast message")
		return
	}

	// at this point, r.commitIndex >= int(rpc.GetCommitIndex())
	if r.commitIndex == int(rpc.GetCommitIndex()) {
		if r.benOrState.benOrIteration < int(rpc.GetIteration()) {
			heardServerFromBroadcast := make([]bool, r.N)
			heardServerFromBroadcast[rpc.GetSenderId()] = true
			// dlog.Println("Replica", r.Id, "hmmm1")
			r.startNewBenOrPlusIteration(int(rpc.GetIteration()), []Entry{rpc.GetBroadcastEntry()}, heardServerFromBroadcast)
		
		} else if r.benOrState.benOrRunning && r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrStage == Broadcasting {
			heardFromServerBefore := r.benOrState.heardServerFromBroadcast[rpc.GetSenderId()]

			if !heardFromServerBefore {
				r.benOrState.benOrBroadcastMsgs = append(r.benOrState.benOrBroadcastMsgs, rpc.GetBroadcastEntry())
				r.benOrState.heardServerFromBroadcast[rpc.GetSenderId()] = true

				if len(r.benOrState.benOrBroadcastMsgs) > r.N/2 {
					haveMajEntry, majEntry := r.getMajorityBroadcastMsg()

					vote := Vote0
					if haveMajEntry {
						if r.commitIndex + 1 < len(r.log) && r.log[r.commitIndex + 1] != r.benOrState.benOrBroadcastEntry {
							vote = Vote0
						} else {
							vote = Vote1
						}
					}

					biasedCoin := false
					if r.commitIndex + 1 < len(r.log) {
						biasedCoin = true
					}

					r.benOrState = BenOrState{
						benOrRunning: true,
					
						benOrIteration: r.benOrState.benOrIteration,
						benOrPhase: r.benOrState.benOrPhase,
						benOrStage: StageOne,
					
						benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
						benOrBroadcastMsgs: r.benOrState.benOrBroadcastMsgs,
						heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
				
						haveMajEntry: haveMajEntry,
						benOrMajEntry: majEntry,
				
						benOrVote: vote,
						benOrConsensusMsgs: make([]uint8, 0),
						heardServerFromConsensus: make([]bool, r.N),

						prevPhaseFinalValue: VoteUninitialized,
						
						biasedCoin: biasedCoin,
					}

					r.benOrState.benOrConsensusMsgs = append(r.benOrState.benOrConsensusMsgs, vote)
					r.benOrState.heardServerFromConsensus[r.Id] = true

					r.startBenOrConsensusStage()
				}
			}
		}
	}

	if t, ok := rpc.(*BenOrBroadcast); ok {
		r.sendBenOrReply(rpc.GetSenderId(), int(t.CommitIndex))
	}
	// dlog.Println("Replica", r.Id, "ending handleBenOrBroadcast", r.benOrStartTimer.active, r.benOrResendTimer.active)
	// otherwise, this is a BenOrBroadcastReply, so we don't need to send anything back to the sender.
	return
}

func (r *Replica) getMajorityBroadcastMsg() (bool, Entry) {
	var numMsgs int = len(r.benOrState.benOrBroadcastMsgs)
	msgs := r.benOrState.benOrBroadcastMsgs

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].ServerId != msgs[j].ServerId {
			return msgs[i].ServerId < msgs[j].ServerId
		}
		if msgs[i].Timestamp != msgs[j].Timestamp {
			return msgs[i].Timestamp < msgs[j].Timestamp
		}
		return msgs[i].Term < msgs[j].Term
	})

	haveMajEntry := false
	majEntry := emptyEntry

	counter := 0
	senderId := int32(-1)
	timestamp := int64(-1)

	for i := 0; i < numMsgs; i++ {
		if msgs[i].ServerId == senderId && msgs[i].Timestamp == timestamp {
			counter++
		} else {
			counter = 1
			senderId = msgs[i].ServerId
			timestamp = msgs[i].Timestamp
		}
		
		if counter > r.N/2 {
			haveMajEntry = true
			majEntry = msgs[i]
			break
		}
	}

	// dlog.Println("Server", r.Id, "haveMajEntry: ", haveMajEntry, " majEntry: ", majEntry, "broadcast messages", r.benOrState.benOrBroadcastMsgs)

	return haveMajEntry, majEntry
}

/************************************** Ben Or Consensus **********************************************/

// only called if actually need to run benOr at this index
func (r *Replica) startBenOrConsensusStage() {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	entries := make([]Entry, 0)
	if r.commitIndex + 1 < len(r.log) {
		entries = r.log[r.commitIndex + 1:]
	}

	args := &BenOrConsensus{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
		Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
		Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
		Entries: entries, PQEntries: r.pq.extractList(),
	}

	// dlog.Println("server", r.Id, "sending BenOrConsensus with vote", r.benOrState.benOrVote, "pq entries: ", len(r.pq.extractList()))
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}
}

func (r *Replica) handleBenOrConsensus(rpc BenOrConsensusMsg) {
	r.handleIncomingTerm(rpc)

	if !r.updateLogIfPossible(rpc) {
		r.sendOneGetCommittedData(int(rpc.GetSenderId()))
		return
	}

	if rpc.GetBenOrMsgValid() == False {
		return
	}

	// at this point, r.commitIndex >= int(rpc.GetCommitIndex())
	if r.commitIndex == int(rpc.GetCommitIndex()) {
		caseIteration := r.benOrState.benOrIteration < int(rpc.GetIteration())
		casePhase := r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrPhase < int(rpc.GetPhase())
		caseStage := r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrPhase == int(rpc.GetPhase()) && r.benOrState.benOrStage < rpc.GetStage()

		if caseIteration || casePhase || caseStage {
			// if !r.benOrState.benOrRunning {
			timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
			setTimer(r.benOrResendTimer, time.Duration(timeout) * time.Millisecond)
			// }

			var benOrBroadcastEntry Entry
			var haveMajEntry bool
			var majEntry Entry
			var biasedCoin bool
			var vote uint8
			if caseIteration {
				benOrIndex := r.commitIndex + 1

				if benOrIndex < len(r.log) {
					// dlog.Println(r.Id, "Ben Or Plus: Broadcasting entry", benOrIndex, "from log")
					// benOrBroadcastEntry = r.log[benOrIndex]
					// benOrBroadcastEntry.Index = int32(benOrIndex)
					biasedCoin = true
				} else if !r.pq.isEmpty() {
					// dlog.Println(r.Id, "Ben Or Plus: Broadcasting entry", benOrIndex, "from pq, current commit indx:", r.commitIndex)
					// benOrBroadcastEntry = r.pq.peek()
					// benOrBroadcastEntry.Index = int32(benOrIndex)
					biasedCoin = false
				}

				benOrBroadcastEntry = emptyEntry
				haveMajEntry = convertIntegerToBool(rpc.GetHaveMajEntry())
				majEntry = rpc.GetMajEntry()

				if rpc.GetStage() == StageOne {
					if rpc.GetPhase() == 0 {
						if haveMajEntry {
							vote = 1
						} else {
							vote = 0
						}
					} else {
						if rpc.GetPrevPhaseFinalValue() == VoteQuestionMark {
							vote = uint8(rand.Intn(2))
						} else {
							vote = rpc.GetVote()
						}
					}
				} else {
					vote = rpc.GetVote()
				}
			}

			if casePhase || caseStage {
				benOrIndex := r.commitIndex + 1

				benOrBroadcastEntry = r.benOrState.benOrBroadcastEntry
				haveMajEntry = r.benOrState.haveMajEntry
				majEntry = r.benOrState.benOrMajEntry

				if convertIntegerToBool(rpc.GetHaveMajEntry()) {
					if !haveMajEntry || r.benOrState.benOrMajEntry.Term < rpc.GetMajEntry().Term {
						majEntry = rpc.GetMajEntry()
					}
					haveMajEntry = true
				}
				biasedCoin = r.benOrState.biasedCoin

				if rpc.GetStage() == StageOne {
					if rpc.GetPhase() == 0 {
						if r.benOrRunning() && haveMajEntry && benOrBroadcastEntry != majEntry {
							vote = 0
						} else {
							if haveMajEntry {
								vote = 1
							} else {
								vote = 0
							}
						}

						if benOrIndex < len(r.log) {
							biasedCoin = true
						} else if !r.pq.isEmpty() {
							biasedCoin = false
						}
					} else {
						if rpc.GetPrevPhaseFinalValue() == VoteQuestionMark {
							vote = uint8(rand.Intn(2))
						} else {
							vote = rpc.GetVote()
						}
					}
				} else {
					vote = rpc.GetVote()
				}
			}

			r.benOrState = BenOrState{
				benOrRunning: true,
			
				benOrIteration: int(rpc.GetIteration()),
				benOrPhase: int(rpc.GetPhase()),
				benOrStage: rpc.GetStage(),
			
				benOrBroadcastEntry: benOrBroadcastEntry,
				benOrBroadcastMsgs: make([]Entry, 0),
				heardServerFromBroadcast: make([]bool, r.N),
			
				haveMajEntry: haveMajEntry,
				benOrMajEntry: majEntry,
		
				benOrVote: vote,
				benOrConsensusMsgs: make([]uint8, 0),
				heardServerFromConsensus: make([]bool, r.N),

				prevPhaseFinalValue: rpc.GetPrevPhaseFinalValue(),
				
				biasedCoin: biasedCoin,
			}

			r.benOrState.benOrConsensusMsgs = append(r.benOrState.benOrConsensusMsgs, vote, rpc.GetVote())
			r.benOrState.heardServerFromConsensus[r.Id] = true
			r.benOrState.heardServerFromConsensus[rpc.GetSenderId()] = true
		}

		if r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrPhase == int(rpc.GetPhase()) && r.benOrState.benOrStage == rpc.GetStage() {
			haveMajEntry := r.benOrState.haveMajEntry
			majEntry := r.benOrState.benOrMajEntry

			if convertIntegerToBool(rpc.GetHaveMajEntry()) {
				if !haveMajEntry || r.benOrState.benOrMajEntry.Term < rpc.GetMajEntry().Term {
					majEntry = rpc.GetMajEntry()
				}
				haveMajEntry = true
			}
			
			r.benOrState.haveMajEntry = haveMajEntry
			r.benOrState.benOrMajEntry = majEntry
			
			heardFromServerBefore := r.benOrState.heardServerFromConsensus[rpc.GetSenderId()]

			if !heardFromServerBefore {
				r.benOrState.benOrConsensusMsgs = append(r.benOrState.benOrConsensusMsgs, rpc.GetVote())
				r.benOrState.heardServerFromConsensus[rpc.GetSenderId()] = true

				if len(r.benOrState.benOrConsensusMsgs) > r.N/2 {
					// haveMajEntry, majEntry := r.getMajorityBroadcastMsg()

					msgs := r.benOrState.benOrConsensusMsgs
	
					voteCount := [3]int{0, 0}
					for _, msg := range msgs {
						voteCount[msg]++
					}

					if r.benOrState.benOrStage == StageTwo {
						if voteCount[0] > 0 && voteCount[1] > 0 {
							log.Fatal("Should not have both votes")
						}
					}

					if r.benOrState.benOrStage == StageOne {
						var vote uint8;
						if voteCount[0] > r.N/2 {
							vote = Vote0
						} else if voteCount[1] > r.N/2 {
							vote = Vote1
						} else {
							vote = VoteQuestionMark
						}

						r.benOrState = BenOrState{
							benOrRunning: true,
						
							benOrIteration: r.benOrState.benOrIteration,
							benOrPhase: r.benOrState.benOrPhase,
							benOrStage: StageTwo,
						
							benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
							benOrBroadcastMsgs: r.benOrState.benOrBroadcastMsgs,
							heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
					
							haveMajEntry: r.benOrState.haveMajEntry,
							benOrMajEntry: r.benOrState.benOrMajEntry,
					
							benOrVote: vote,
							benOrConsensusMsgs: make([]uint8, 0),
							heardServerFromConsensus: make([]bool, r.N),

							prevPhaseFinalValue: r.benOrState.prevPhaseFinalValue,
							
							biasedCoin: r.benOrState.biasedCoin,
						}
	
						r.benOrState.benOrConsensusMsgs = append(r.benOrState.benOrConsensusMsgs, vote)
						r.benOrState.heardServerFromConsensus[r.Id] = true

						r.startBenOrConsensusStage()
					} else if r.benOrState.benOrStage == StageTwo {
						// dlog.Println("Consensus reached on server", r.Id, "for stage 2")
						if voteCount[0] > r.N/2 {
							// dlog.Println("Replica", r.Id, "hmmm2")
							r.startNewBenOrPlusIteration(r.benOrState.benOrIteration + 1, make([]Entry, 0), make([]bool, r.N))
						} else if voteCount[1] > r.N/2 {
							// commit entry
							// fmt.Println("Iteration: ", r.benOrState.benOrIteration)

							oldCommitIndex := r.commitIndex
							newCommittedEntry := r.benOrState.benOrMajEntry
							benOrIndex := r.commitIndex + 1
							potentialEntries := make([]Entry, 0)
							// dlog.Println("Committing entry (using ben or)", newCommittedEntry, "on server", r.Id, "log is", logToString(r.log), "for iteration", r.benOrState.benOrIteration)
							r.benOrState = emptyBenOrState

							if benOrIndex < len(r.log) {
								if !entryEqual(r.log[benOrIndex], newCommittedEntry) {
									// dlog.Println("Replica", r.Id, "not equal", r.log[benOrIndex], newCommittedEntry)
									for i := benOrIndex; i < len(r.log); i++ {
										r.inLog.remove(r.log[i])
										potentialEntries = append(potentialEntries, r.log[i])
									}
									r.log = r.log[:benOrIndex]

									r.inLog.add(newCommittedEntry)
									r.log = append(r.log, newCommittedEntry)
									r.pq.remove(newCommittedEntry)
								}
							} else {
								r.inLog.add(newCommittedEntry)
								r.log = append(r.log, newCommittedEntry)
								r.pq.remove(newCommittedEntry)
							}

							for _, v := range(potentialEntries) {
								if !r.seenBefore(v) { r.pq.push(v) }
							}

							if r.leaderState.isLeader {
								for !r.pq.isEmpty() {
									entry := r.pq.pop()
									entry.Term = int32(r.term)
									entry.Index = int32(len(r.log))
						
									if !r.inLog.contains(entry) {
										r.log = append(r.log, entry)
										r.inLog.add(entry)
									}
								}
						
								for i := 0; i < r.N; i++ {
									// if i != int(r.Id) || i != int(rpc.GetSenderId()) {
									if i != int(r.Id) {
										r.leaderState.repNextIndex[i] = min(r.leaderState.repNextIndex[i], oldCommitIndex + 1)
										r.leaderState.repMatchIndex[i] = min(r.leaderState.repMatchIndex[i], oldCommitIndex)
									}
						
									r.leaderState.repNextIndex[r.Id] = len(r.log)
									r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
								}
							}
							
							r.commitIndex++

							if r.leaderState.isLeader {
								clearTimer(r.benOrResendTimer)

							} else {
								timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
								setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)

								clearTimer(r.benOrResendTimer)
							}

						} else {
							var prevPhaseFinalValue uint8
							var initialVote uint8
							if voteCount[0] > 0 {
								prevPhaseFinalValue = Vote0
								initialVote = Vote0
							} else if voteCount[1] > 0 {
								prevPhaseFinalValue = Vote1
								initialVote = Vote1
							} else {
								prevPhaseFinalValue = VoteQuestionMark
								if r.benOrState.biasedCoin {
									initialVote = Vote0
									// dlog.Println("Replica", r.Id, "biased coin flip for phase", r.benOrState.benOrPhase + 1)
								} else {
									initialVote = uint8(rand.Intn(2))
									// dlog.Println("Replica", r.Id, "random coin flip for phase", r.benOrState.benOrPhase + 1, "flip is", initialVote)
								}
							}

							r.benOrState = BenOrState{
								benOrRunning: true,
							
								benOrIteration: r.benOrState.benOrIteration,
								benOrPhase: r.benOrState.benOrPhase + 1,
								benOrStage: StageOne,
							
								benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
								benOrBroadcastMsgs: r.benOrState.benOrBroadcastMsgs,
								heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
						
								haveMajEntry: r.benOrState.haveMajEntry,
								benOrMajEntry: r.benOrState.benOrMajEntry,
						
								benOrVote: initialVote,
								benOrConsensusMsgs: make([]uint8, 0),
								heardServerFromConsensus: make([]bool, r.N),

								prevPhaseFinalValue: prevPhaseFinalValue,
								
								biasedCoin: r.benOrState.biasedCoin,
							}
		
							r.benOrState.benOrConsensusMsgs = append(r.benOrState.benOrConsensusMsgs, initialVote)
							r.benOrState.heardServerFromConsensus[r.Id] = true
	
							r.startBenOrConsensusStage()
						}
					}
				}
			}
		}
	}

	if t, ok := rpc.(*BenOrConsensus); ok {
		r.sendBenOrReply(rpc.GetSenderId(), int(t.GetCommitIndex()))
	}
	// otherwise, this is a BenOrBroadcastReply, so we don't need to send anything back to the sender.
	return
}

/************************************** BenOr Reply **********************************************/
func (r *Replica) sendBenOrReply(senderId int32, rpcCommitIndex int) {
	entries := make([]Entry, 0)
	if rpcCommitIndex + 1 < len(r.log) {
		entries = r.log[rpcCommitIndex + 1:]
	}

	if r.benOrState.benOrStage == NotRunning || r.benOrState.benOrStage == Broadcasting {
		args := &BenOrBroadcastReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			BenOrMsgValid: convertBoolToInteger(r.benOrState.benOrRunning), Iteration: int32(r.benOrState.benOrIteration), BroadcastEntry: r.benOrState.benOrBroadcastEntry,
			StartIndex: int32(rpcCommitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		// dlog.Println("replica", r.Id, "sending BenOrBroadcastReply to", senderId, "with", len(entries), "entries and", "pq entries: ", len(r.pq.extractList()))
		// dlog.Println("replica", r.Id, "sending BenOrBroadcastReply to", senderId, "with log:", logToString(r.log), "pq entries:", logToString(r.pq.extractList()), "broadcast entry:", r.benOrState.benOrBroadcastEntry, "majEntry", r.benOrState.benOrMajEntry)
		r.SendMsg(senderId, r.benOrBroadcastReplyRPC, args)

	} else if r.benOrState.benOrStage == StageOne || r.benOrState.benOrStage == StageTwo {
		args := &BenOrConsensusReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			BenOrMsgValid: True, Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
			Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
			StartIndex: int32(rpcCommitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		// dlog.Println("replica", r.Id, "sending BenOrConsensusReply to", senderId, "with", len(entries), "entries and", "pq entries: ", len(r.pq.extractList()))
		// dlog.Println("replica", r.Id, "sending BenOrConsensusReply to", senderId, "with log:", logToString(r.log), "pq entries:", logToString(r.pq.extractList()), "vote:", r.benOrState.benOrVote, "haveMajEntry:", r.benOrState.haveMajEntry, "majEntry:", r.benOrState.benOrMajEntry)
		r.SendMsg(senderId, r.benOrConsensusReplyRPC, args)
	}
}

/************************************** Resend Ben Or **********************************************/

func (r *Replica) resendBenOrTimer() {
	timeout := rand.Intn(r.benOrResendTimeout/2) + r.benOrResendTimeout/2
	setTimer(r.benOrResendTimer, time.Duration(timeout)*time.Millisecond)

	entries := make([]Entry, 0)
	if int(r.commitIndex) + 1 < len(r.log) {
		entries = r.log[r.commitIndex + 1:]
	}

	if !r.benOrState.benOrRunning {
		log.Fatal("Resending BenOr when not running")
	}

	if r.benOrState.benOrStage == Broadcasting {
		// dlog.Println("replica", r.Id, "resending BenOrBroadcast", time.Now().UnixMilli(), "with", len(entries), "entries and", "pq entries: ", len(r.pq.extractList()))
		args := &BenOrBroadcast{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			Iteration: int32(r.benOrState.benOrIteration), BroadcastEntry: r.benOrState.benOrBroadcastEntry,
			StartIndex: int32(r.commitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	}

	if r.benOrState.benOrStage == StageOne || r.benOrState.benOrStage == StageTwo {
		// dlog.Println("replica", r.Id, "resending BenOrConsensus Stage", r.benOrState.benOrStage, time.Now().UnixMilli(), "with", len(entries), "entries and", "pq entries: ", len(r.pq.extractList()))
		args := &BenOrConsensus{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LeaderTerm: int32(r.leaderTerm), LogLength: int32(len(r.log)),
			Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
			Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
			Entries: entries, PQEntries: r.pq.extractList(),
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrConsensusRPC, args)
			}
		}
	}
}