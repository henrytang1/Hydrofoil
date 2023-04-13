package randomizedpaxos

import (
	"log"
	"math/rand"
	"sort"
	"time"
)

/************************************** Ben Or Braodcast **********************************************/
func (r *Replica) startBenOrPlus() {
	r.startNewBenOrPlusIteration(0, make([]Entry, 0), make([]bool, r.N))
}

func (r *Replica) startNewBenOrPlusIteration(iteration int, benOrBroadcastMsg []Entry, heardServerFromBroadcast []bool) {
	// if iteration != 0 {
	// 	if !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
	// 		r.pq.push(r.benOrState.benOrBroadcastEntry)
	// 	}
	// }

	var broadcastEntry Entry
	benOrIndex := r.commitIndex + 1
	if benOrIndex < len(r.log) {
		broadcastEntry = r.log[benOrIndex]
		broadcastEntry.Index = int32(r.commitIndex)+1
	} else if !r.pq.isEmpty() {
		broadcastEntry = r.pq.peek()
		broadcastEntry.Index = int32(r.commitIndex)+1
	} else if iteration == 0 {
		// nothing to run ben-or on right now, wait until timeout again
		timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
		setTimer(r.benOrStartTimer, time.Duration(timeout) * time.Millisecond)
	} else {
		log.Fatal("Ben Or Plus: No entry to broadcast")
	}

	heardServerFromBroadcast[r.Id] = true

	r.benOrState = BenOrState{
		benOrRunning: true,
	
		benOrIteration: iteration,
		benOrPhase: 0,
		benOrStage: Broadcasting,
	
		benOrBroadcastEntry: broadcastEntry,
		benOrBroadcastMessages: append(benOrBroadcastMsg, broadcastEntry),
		heardServerFromBroadcast: heardServerFromBroadcast,

		haveMajEntry: false,
		benOrMajEntry: emptyEntry,

		benOrVote: VoteUninitialized,
		benOrConsensusMessages: make([]uint8, 0),
		heardServerFromConsensus: make([]bool, r.N),
		
		biasedCoin: false,
	}

	r.startBenOrBroadcast()
}

func (r *Replica) startBenOrBroadcast() {
	entries := make([]Entry, 0)
	if r.commitIndex + 1 < len(r.log) {
		entries = r.log[r.commitIndex + 1:]
	}

	args := &BenOrBroadcast{
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
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
	if r.isLogMoreUpToDate(rpc) == LowerOrder {
		if r.commitIndex + 1 >= int(rpc.GetStartIndex()) {
			r.updateLogFromRPC(rpc)
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
	}
	return true
}

func (r *Replica) handleBenOrBroadcast(rpc BenOrBroadcastMsg) {
	r.handleIncomingTerm(rpc)

	if !r.updateLogIfPossible(rpc) {
		return
	}

	// at this point, r.commitIndex >= int(rpc.GetCommitIndex())
	if r.commitIndex == int(rpc.GetCommitIndex()) && (r.benOrState.benOrRunning || rpc.GetIteration() > 0) {
		if r.benOrState.benOrIteration < int(rpc.GetIteration()) {
			// if r.benOrState.benOrBroadcastEntry != emptyEntry && !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
			// 	if r.leaderState.isLeader {
			// 		entry := r.benOrState.benOrBroadcastEntry
			// 		entry.Term = int32(r.term)
			// 		entry.Index = int32(len(r.log))
	
			// 		if !r.inLog.contains(entry) {
			// 			r.log = append(r.log, entry)
			// 			r.inLog.add(entry)
			// 		}
			// 		r.leaderState.repNextIndex[r.Id] = len(r.log)
			// 		r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
			// 	} else {
			// 		r.pq.push(r.benOrState.benOrBroadcastEntry)
			// 	}
			// }
			heardServerFromBroadcast := make([]bool, r.N)
			heardServerFromBroadcast[rpc.GetSenderId()] = true
			r.startNewBenOrPlusIteration(int(rpc.GetIteration()), []Entry{rpc.GetBroadcastEntry()}, heardServerFromBroadcast)
		} else if r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrStage == Broadcasting {
			heardFromServerBefore := r.benOrState.heardServerFromBroadcast[rpc.GetSenderId()]

			if !heardFromServerBefore {
				r.benOrState.benOrBroadcastMessages = append(r.benOrState.benOrBroadcastMessages, rpc.GetBroadcastEntry())
				r.benOrState.heardServerFromBroadcast[rpc.GetSenderId()] = true

				if len(r.benOrState.benOrBroadcastMessages) > r.N/2 {
					haveMajEntry, majEntry := r.getMajorityBroadcastMsg()

					vote := Vote0
					if r.benOrState.haveMajEntry {
						vote = Vote1
					}

					r.benOrState = BenOrState{
						benOrRunning: true,
					
						benOrIteration: r.benOrState.benOrIteration,
						benOrPhase: r.benOrState.benOrPhase,
						benOrStage: StageOne,
					
						benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
						benOrBroadcastMessages: r.benOrState.benOrBroadcastMessages,
						heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
				
						haveMajEntry: haveMajEntry,
						benOrMajEntry: majEntry,
				
						benOrVote: vote,
						benOrConsensusMessages: make([]uint8, 0),
						heardServerFromConsensus: make([]bool, r.N),
						
						biasedCoin: false,
					}

					r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, vote)
					r.benOrState.heardServerFromConsensus[r.Id] = true

					r.startBenOrConsensusStage()
				}
			}
		}
	}

	if t, ok := rpc.(*BenOrBroadcast); ok {
		r.sendBenOrReply(rpc.GetSenderId(), int(t.CommitIndex))
	}
	// otherwise, this is a BenOrBroadcastReply, so we don't need to send anything back to the sender.
	return
}

func (r *Replica) getMajorityBroadcastMsg() (bool, Entry) {
	var numMsgs int = len(r.benOrState.benOrBroadcastMessages)
	msgs := r.benOrState.benOrBroadcastMessages

	sort.Slice(msgs, func(i, j int) bool {
		if msgs[i].SenderId != msgs[j].SenderId {
			return msgs[i].SenderId < msgs[j].SenderId
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
		if msgs[i].SenderId == senderId && msgs[i].Timestamp == timestamp {
			counter++
		} else {
			counter = 1
			senderId = msgs[i].SenderId
			timestamp = msgs[i].Timestamp
		}
		
		if counter > r.N/2 {
			haveMajEntry = true
			majEntry = msgs[i]
			break
		}
	}

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
		SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
		Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
		Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
		Entries: entries, PQEntries: r.pq.extractList(),
	}
	// r.SendMsg(r.Id, r.benOrBroadcastRPC, args)
	for i := 0; i < r.N; i++ {
		if int32(i) != r.Id {
			r.SendMsg(int32(i), r.benOrConsensusRPC, args)
		}
	}
}

func (r *Replica) handleBenOrConsensus(rpc BenOrConsensusMsg) {
	r.handleIncomingTerm(rpc)

	if !r.updateLogIfPossible(rpc) {
		return
	}

	// at this point, r.commitIndex >= int(rpc.GetCommitIndex())
	if r.commitIndex == int(rpc.GetCommitIndex()) {
		caseIteration := r.benOrState.benOrIteration < int(rpc.GetIteration())
		casePhase := r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrPhase < int(rpc.GetPhase())
		caseStage := r.benOrState.benOrIteration == int(rpc.GetIteration()) && r.benOrState.benOrPhase == int(rpc.GetPhase()) && r.benOrState.benOrStage < rpc.GetStage()

		if caseIteration || casePhase || caseStage {
			var benOrBroadcastEntry Entry
			var haveMajEntry bool
			var majEntry Entry
			var biasedCoin bool	
			if caseIteration {
				// if r.benOrState.benOrBroadcastEntry != emptyEntry && !r.seenBefore(r.benOrState.benOrBroadcastEntry) {
				// 	if r.leaderState.isLeader {
				// 		entry := r.benOrState.benOrBroadcastEntry
				// 		entry.Term = int32(r.term)
				// 		entry.Index = int32(len(r.log))
		
				// 		if !r.inLog.contains(entry) {
				// 			r.log = append(r.log, entry)
				// 			r.inLog.add(entry)
				// 		}
				// 		r.leaderState.repNextIndex[r.Id] = len(r.log)
				// 		r.leaderState.repMatchIndex[r.Id] = len(r.log) - 1
				// 	} else {
				// 		r.pq.push(r.benOrState.benOrBroadcastEntry)
				// 	}
				// }
				benOrBroadcastEntry = emptyEntry
				haveMajEntry = convertIntegerToBool(rpc.GetHaveMajEntry())
				majEntry = rpc.GetMajEntry()
				biasedCoin = false
			}

			if casePhase || caseStage {
				benOrBroadcastEntry = r.benOrState.benOrBroadcastEntry
				haveMajEntry = r.benOrState.haveMajEntry
				majEntry = r.benOrState.benOrMajEntry

				benOrBroadcastEntry = r.benOrState.benOrBroadcastEntry
				if convertIntegerToBool(rpc.GetHaveMajEntry()) {
					if !haveMajEntry || r.benOrState.benOrMajEntry.Term < rpc.GetMajEntry().Term {
						majEntry = rpc.GetMajEntry()
					}
					haveMajEntry = true
				}
				biasedCoin = r.benOrState.biasedCoin
			}

			r.benOrState = BenOrState{
				benOrRunning: true,
			
				benOrIteration: int(rpc.GetIteration()),
				benOrPhase: int(rpc.GetPhase()),
				benOrStage: rpc.GetStage(),
			
				benOrBroadcastEntry: benOrBroadcastEntry,
				benOrBroadcastMessages: make([]Entry, 0),
				heardServerFromBroadcast: make([]bool, r.N),
			
				haveMajEntry: haveMajEntry,
				benOrMajEntry: majEntry,
		
				benOrVote: rpc.GetVote(),
				benOrConsensusMessages: make([]uint8, 0),
				heardServerFromConsensus: make([]bool, r.N),
				
				biasedCoin: biasedCoin,
			}

			r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, r.benOrState.benOrVote, rpc.GetVote())
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
				r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, rpc.GetVote())
				r.benOrState.heardServerFromConsensus[rpc.GetSenderId()] = true

				if len(r.benOrState.benOrConsensusMessages) > r.N/2 {
					// haveMajEntry, majEntry := r.getMajorityBroadcastMsg()

					msgs := r.benOrState.benOrConsensusMessages
	
					voteCount := [3]int{0, 0}
					for _, msg := range msgs {
						voteCount[msg]++
					}

					var vote uint8;
					if voteCount[0] > r.N/2 {
						vote = Vote0
					} else if voteCount[1] > r.N/2 {
						vote = Vote1
					} else {
						vote = VoteQuestionMark
					}

					if r.benOrState.benOrStage == StageTwo {
						if voteCount[0] > 0 && voteCount[1] > 0 {
							log.Fatal("Should not have both votes")
						}
					}

					if r.benOrState.benOrStage == StageOne {
						r.benOrState = BenOrState{
							benOrRunning: true,
						
							benOrIteration: r.benOrState.benOrIteration,
							benOrPhase: r.benOrState.benOrPhase,
							benOrStage: StageTwo,
						
							benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
							benOrBroadcastMessages: r.benOrState.benOrBroadcastMessages,
							heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
					
							haveMajEntry: r.benOrState.haveMajEntry,
							benOrMajEntry: r.benOrState.benOrMajEntry,
					
							benOrVote: vote,
							benOrConsensusMessages: make([]uint8, 0),
							heardServerFromConsensus: make([]bool, r.N),
							
							biasedCoin: r.benOrState.biasedCoin,
						}
	
						r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, vote)
						r.benOrState.heardServerFromConsensus[r.Id] = true

						r.startBenOrConsensusStage()
					} else if r.benOrState.benOrStage == StageTwo {
						if vote == Vote0 {
							r.startNewBenOrPlusIteration(r.benOrState.benOrIteration + 1, make([]Entry, 0), make([]bool, r.N))
						} else if vote == Vote1 {
							// commit entry
							oldCommitIndex := r.commitIndex
							newCommittedEntry := r.benOrState.benOrMajEntry
							benOrIndex := r.commitIndex + 1
							potentialEntries := make([]Entry, 0)
							r.benOrState = emptyBenOrState

							if benOrIndex < len(r.log) && !entryEqual(r.log[benOrIndex], newCommittedEntry) {
								for i := benOrIndex; i < len(r.log); i++ {
									r.inLog.remove(r.log[i])
									potentialEntries = append(potentialEntries, r.log[i])
								}
								r.log = r.log[:benOrIndex]

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
						
									// firstEntryIndex := int(rpc.GetStartIndex())
									// r.leaderState.repNextIndex[rpc.GetSenderId()] = firstEntryIndex + len(rpc.GetEntries())
									// r.leaderState.repMatchIndex[rpc.GetSenderId()] = firstEntryIndex + len(rpc.GetEntries()) - 1
								}
							}
							
							r.commitIndex++
							clearTimer(r.benOrResendTimer)

							// restart BenOr Timer
							timeout := rand.Intn(r.benOrStartTimeout/2) + r.benOrStartTimeout/2
							setTimer(r.benOrStartTimer, time.Duration(timeout)*time.Millisecond)
						} else {
							var initialVote uint8
							if r.benOrState.biasedCoin {
								initialVote = 0
							} else {
								initialVote = uint8(rand.Intn(2))
							}

							r.benOrState = BenOrState{
								benOrRunning: true,
							
								benOrIteration: r.benOrState.benOrIteration,
								benOrPhase: r.benOrState.benOrPhase + 1,
								benOrStage: StageOne,
							
								benOrBroadcastEntry: r.benOrState.benOrBroadcastEntry,
								benOrBroadcastMessages: r.benOrState.benOrBroadcastMessages,
								heardServerFromBroadcast: r.benOrState.heardServerFromBroadcast,
						
								haveMajEntry: r.benOrState.haveMajEntry,
								benOrMajEntry: r.benOrState.benOrMajEntry,
						
								benOrVote: initialVote,
								benOrConsensusMessages: make([]uint8, 0),
								heardServerFromConsensus: make([]bool, r.N),
								
								biasedCoin: r.benOrState.biasedCoin,
							}
		
							r.benOrState.benOrConsensusMessages = append(r.benOrState.benOrConsensusMessages, vote)
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
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			BenOrMsgValid: convertBoolToInteger(r.benOrState.benOrRunning), Iteration: int32(r.benOrState.benOrIteration), BroadcastEntry: r.benOrState.benOrBroadcastEntry,
			StartIndex: int32(rpcCommitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		r.SendMsg(senderId, r.benOrBroadcastReplyRPC, args)
	}

	if r.benOrState.benOrStage == StageOne || r.benOrState.benOrStage == StageTwo {
		args := &BenOrConsensusReply{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			BenOrMsgValid: True, Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
			Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
			StartIndex: int32(rpcCommitIndex) + 1, Entries: entries, PQEntries: r.pq.extractList(),
		}
		r.SendMsg(senderId, r.benOrBroadcastReplyRPC, args)
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
		args := &BenOrBroadcast{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
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
		args := &BenOrConsensus{
			SenderId: r.Id, Term: int32(r.term), CommitIndex: int32(r.commitIndex), LogTerm: int32(r.logTerm), LogLength: int32(len(r.log)),
			Iteration: int32(r.benOrState.benOrIteration), Phase: int32(r.benOrState.benOrPhase), Stage: r.benOrState.benOrStage,
			Vote: r.benOrState.benOrVote, HaveMajEntry: convertBoolToInteger(r.benOrState.haveMajEntry), MajEntry: r.benOrState.benOrMajEntry,
			Entries: entries, PQEntries: r.pq.extractList(),
		}
		for i := 0; i < r.N; i++ {
			if int32(i) != r.Id {
				r.SendMsg(int32(i), r.benOrBroadcastRPC, args)
			}
		}
	}
}