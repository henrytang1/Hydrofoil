numRuns=10
for ((i=0; i<numRuns; i++))
do
        echo $i
        (./bin/master -N 5) &
        sleep 3
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7070 -hydrofoil=true -exec=true -dreply=true) &> log0.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7071 -hydrofoil=true -exec=true -dreply=true) &> log1.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7072 -hydrofoil=true -exec=true -dreply=true) &> log2.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7073 -hydrofoil=true -exec=true -dreply=true) &> log3.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7074 -hydrofoil=true -exec=true -dreply=true) &> log4.txt &
        sleep 3
        (./bin/clientmain -maddr=127.0.0.1 -mport=7087 -oneLeaderFailure=true -hasALeader=true -id=$i -tput_interval_in_sec=0.02 -q=16000) &
        sleep 25
        bash killall.sh
        sleep 2
done

for ((i=numRuns; i<2*numRuns; i++))
do
        echo $i
        (./bin/master -N 5) &
        sleep 3
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7070 -raft=true -exec=true -dreply=true) &> log5.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7071 -raft=true -exec=true -dreply=true) &> log6.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7072 -raft=true -exec=true -dreply=true) &> log7.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7073 -raft=true -exec=true -dreply=true) &> log8.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7074 -raft=true -exec=true -dreply=true) &> log9.txt &
        sleep 3
        (./bin/clientmain -maddr=127.0.0.1 -mport=7087 -oneLeaderFailure=true -hasALeader=true -id=$i -tput_interval_in_sec=0.02 -q=16000) &
        sleep 25
        bash killall.sh
        sleep 2
done

for ((i=2*numRuns; i<3*numRuns; i++))
do
        echo $i
        (./bin/master -N 5) &
        sleep 3
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7070 -benor=true -exec=true -dreply=true) &> log10.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7071 -benor=true -exec=true -dreply=true) &> log11.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7072 -benor=true -exec=true -dreply=true) &> log12.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7073 -benor=true -exec=true -dreply=true) &> log13.txt &
        (./bin/server -maddr=127.0.0.1 -mport=7087 -addr=127.0.0.1 -port=7074 -benor=true -exec=true -dreply=true) &> log14.txt &
        sleep 3
        (./bin/clientmain -maddr=127.0.0.1 -mport=7087 -oneLeaderFailure=true -hasALeader=false -id=$i -tput_interval_in_sec=0.02 -q=16000) &
        sleep 25
        bash killall.sh
        sleep 2
done
