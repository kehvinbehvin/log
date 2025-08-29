## Memory profile
- go run . -memprofile mem.prof
- go tool pprof -inuse_space mem.prof
- go tool pprof mem.prof
- GODEBUG=gctrace=1 go run .

## CPU profile
- go run . -cpuprofile cpu.prof
- go tool pprof cpu.prof

Created them separately so that they don't interfere with each other

## Observations
Worker: Process the ingested log line
Writer: Writers the log line to the output file

At 1:1 Worker:Writer, the program used 1.6 GB of memory. We reduced it to 220MB by []rune
instead of allocating temporary strings. 

This memory size is expected because the raw log file is 166 MB, We were allocating a 
new string when reading the file (scanner.Text()) and when passing the string as an 
argument to the writer to write to file. So each log line is copied twice, so the 
program memory will grow 2X of the size of the data coming in. 

Increasing to 2:2 Worker:Writer, the program ran using 206.41MB memory
Increasing to 8:8 Worker:Writer, the program ran using 258.09MB memory

As expected, without dealing with the scanner.Text, we would see any significant improvement
with memory usage which accounted for 74.82% of memory usage (193.11MB). Interestingly,
main.BatchOutput only accounted for 8.52% of memory usage (22MB), which is unexpected.

Even at 1:1 worker:writer, main.BatchOutput only accounted for 7% of memory usage (16MB)

Replaced scanner.Text with scanner.Bytes which does not allocate a new string.
Program memory reduced to 163.49MB. Instead of expecting 27MB, we swapped out memory 
usage in scanner.Text for more memory usage by main.startWorker (80 MB) and main.BatchOutput (80 MB)

BatchOutput memory usage increased 10x with this simple change.

An error resetting the runeBuf, resulted in a huge memory usage. We were reassigning runeBuf in Startworker
instead of reusing the allocated memory. Making the adjustment resulted in a drop in the program memory usaged
Total: 62 MB, startworker (32MB) and BatchOuput (23MB).

Still not 27 MB as expected, investigation continues.

Investigations
(pprof) list startWorker
Total: 62.45MB
ROUTINE ======================== main.(*LogNormaliser).startWorker in /Users/kevin/Documents/Kevin/golang-sd/main.go
      32MB       32MB (flat, cum) 51.25% of Total
         .          .    300:func (ln *LogNormaliser) startWorker(id int, lines <-chan []rune, result chan<- string, wg *sync.WaitGroup) error {
         .          .    301:   defer wg.Done()
         .          .    302:   fmt.Println("Worker started")
         .          .    303:
         .          .    304:   var runeBuf []rune
         .          .    305:
         .          .    306:   for line := range lines {
         .          .    307:           // Reset rune buffer
         .          .    308:           runeBuf = runeBuf[:0]
         .          .    309:           runeBuf = append(runeBuf, line...)
         .          .    310:
         .          .    311:           logLine, err := ProcessLogLine(runeBuf)
         .          .    312:           if err != nil {
         .          .    313:                   return err
         .          .    314:           }
         .          .    315:
         .          .    316:           ln.performance.RecordLineProcessed()
         .          .    317:
         .          .    318:           // Blocks if channel is full
      32MB       32MB    319:           result <- string(logLine)
         .          .    320:   }
         .          .    321:
         .          .    322:   return nil
         .          .    323:}
         .          .    324:


Total: 62.45MB
ROUTINE ======================== main.BatchOutput in /Users/kevin/Documents/Kevin/golang-sd/main.go
      23MB       23MB (flat, cum) 36.83% of Total
         .          .    123:func BatchOutput(results <-chan string, wg *sync.WaitGroup, outputFile *os.File) {
         .          .    124:   defer wg.Done()
         .          .    125:
         .          .    126:   // Defaults 4KB per batch
         .          .    127:   writer := bufio.NewWriter(outputFile)
         .          .    128:   defer writer.Flush()
         .          .    129:
         .          .    130:   for result := range results {
      23MB       23MB    131:           writer.WriteString(result + "\n")
         .          .    132:   }
         .          .    133:}
         .          .    134:
         .          .    135:type PerformanceMetrics struct {
         .          .    136:   totalLinesProcessed int64

It is clear from this investigation that string() is allocating memory for a new string when passing to the channel.
It it likely that the GC is cleaning up the references after being written to the output file

File: kevin
Type: cpu
Time: 2025-08-18 14:55:38 PDT
Duration: 1.10s, Total samples = 4.21s (381.92%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 4.12s, 97.86% of 4.21s total
Dropped 31 nodes (cum <= 0.02s)
Showing top 10 nodes out of 53
      flat  flat%   sum%        cum   cum%
     2.06s 48.93% 48.93%      2.06s 48.93%  runtime.usleep
     1.18s 28.03% 76.96%      1.18s 28.03%  runtime.pthread_cond_wait
     0.62s 14.73% 91.69%      0.62s 14.73%  syscall.syscall
     0.12s  2.85% 94.54%      0.12s  2.85%  runtime.pthread_cond_signal
     0.04s  0.95% 95.49%      0.67s 15.91%  main.(*LogNormaliser).Start
     0.04s  0.95% 96.44%      0.08s  1.90%  main.Mask
     0.03s  0.71% 97.15%      0.03s  0.71%  main.RemoveAlphanumeric
     0.03s  0.71% 97.86%      0.03s  0.71%  runtime.mapaccess2_fast32
         0     0% 97.86%      0.58s 13.78%  bufio.(*Scanner).Scan
         0     0% 97.86%      0.03s  0.71%  bufio.(*Writer).Flush

(pprof) peek runtime.usleep
Showing nodes accounting for 4.21s, 100% of 4.21s total
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context              
----------------------------------------------------------+-------------
                                             1.92s 93.20% |   runtime.runqgrab
                                             0.14s  6.80% |   runtime.osyield
     2.06s 48.93% 48.93%      2.06s 48.93%                | runtime.usleep
----------------------------------------------------------+-------------

60-70% of the cpu usage is related to GC indicating high GC pressure.


Optimisation: 
- Remove string allocation and use rune slice for results channel
- Write rune slices directly into files

Results:
File: kevin
Type: alloc_space
Time: 2025-08-18 15:06:22 PDT
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 7299.72kB, 100% of 7299.72kB total
Showing top 10 nodes out of 14
      flat  flat%   sum%        cum   cum%
 5760.72kB 78.92% 78.92%  5760.72kB 78.92%  main.(*LogNormaliser).Start
    1539kB 21.08%   100%     1539kB 21.08%  runtime.allocm
         0     0%   100%  5760.72kB 78.92%  main.main
         0     0%   100%  5760.72kB 78.92%  runtime.main
         0     0%   100%      513kB  7.03%  runtime.mcall
         0     0%   100%     1026kB 14.06%  runtime.mstart
         0     0%   100%     1026kB 14.06%  runtime.mstart0
         0     0%   100%     1026kB 14.06%  runtime.mstart1
         0     0%   100%     1539kB 21.08%  runtime.newm
         0     0%   100%      513kB  7.03%  runtime.park_m
(pprof) exit

Program now runs on 7MB of memory. Significant improvement in memory usage.
More than expected. 


Optimisation: Reduced size of channel buffer to 10000 from 100000.
Program memory reduced to 1025.69kB.

Previously 2.5MB allocation of memory due to channel buffer is now gone.

Noticed that StartWorker was reallocating quite alot, due to the dynamic log sizing.
Changed it to a fix size of 10000. This reduced reallocation to very low, but 
this increased the total memory allocated to the program by 3 times. now at 3.7 MB

Turns out there was no GC pressure. Most of the usleep was due to goroutines waiting around for work. inuse_space profile
shows only 1.5MB total heap usage, if GC pressure was an issue, the heap would be much bigger, considering that
the total size allocated to the program was only 2MB in total.

Tested 3 scenarios,
1 Worker : 4 Writers
4 Workers : 1 Writer
4 Workers : 4 Writers

In all 3 scenarios, we encountered similar throughput, around 800 ms for all 500K logs, with 1:4 ratio with a slight dip in usleep, with
no further improvments after increasing to 1:8 ratio. Indicating that Workers are processing faster than Writers at around 4 times faster. But ultimately, both workers and writers are processing faster than the I/O when reading the file. 

It should be noted that decreasing the channel buffer from 10000 to 50, did not result in any channel back pressure. Further strengthening the hypothesis that we're currently I/O bound.

Optimisation: increase scanner read from default 64Kb to 1MB.
Observation: no change in duration, slight increase in program memory to 2.7MB

Conclusion, scanner.Bytes does not increase throughput even with a larger buffer because it scanners the logs line by line. So 
we're bound by the speed of scanner.Scan

 golang-sd git:(master) ✗ GODEBUG=gctrace=1 go run .
gc 1 @0.028s 2%: 0.73+0.72+0.10 ms clock, 7.3+1.1/1.0/0+1.0 ms cpu, 3->4->0 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 2 @0.035s 3%: 0.15+1.8+0.52 ms clock, 1.5+0.41/0.69/0+5.2 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 3 @0.039s 4%: 0.069+0.48+0.068 ms clock, 0.69+0.51/0.99/0.29+0.68 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 4 @0.041s 4%: 0.014+0.92+0.032 ms clock, 0.14+0.061/0.80/1.0+0.32 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 5 @0.074s 2%: 0.19+0.60+0.016 ms clock, 1.9+0.30/1.4/0.24+0.16 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 6 @0.077s 3%: 0.047+0.59+0.038 ms clock, 0.47+1.1/1.3/0.88+0.38 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 7 @0.089s 2%: 0.038+0.43+0.007 ms clock, 0.38+0.45/1.0/0.61+0.076 ms cpu, 3->3->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 8 @0.091s 3%: 0.050+0.52+0.006 ms clock, 0.50+0.30/1.1/0.48+0.068 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 9 @0.094s 3%: 0.037+0.35+0.004 ms clock, 0.37+0/0.91/1.1+0.041 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 10 @0.098s 3%: 0.033+0.38+0.006 ms clock, 0.33+0.14/0.87/1.0+0.062 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 11 @0.106s 2%: 0.045+0.32+0.011 ms clock, 0.45+0/0.85/1.4+0.11 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 12 @0.117s 2%: 0.034+0.40+0.005 ms clock, 0.34+0/0.94/1.2+0.053 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 13 @0.123s 2%: 0.039+0.39+0.016 ms clock, 0.39+0/0.97/1.3+0.16 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 14 @0.127s 2%: 0.025+0.47+0.027 ms clock, 0.25+0.062/0.96/0.68+0.27 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 15 @0.129s 2%: 0.031+0.50+0.026 ms clock, 0.31+0.14/1.1/0.64+0.26 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 16 @0.133s 2%: 0.024+0.32+0.003 ms clock, 0.24+0.11/0.80/0.95+0.037 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 17 @0.136s 2%: 0.033+0.50+0.003 ms clock, 0.33+0.14/1.0/0.93+0.039 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 18 @0.142s 2%: 0.038+0.30+0.008 ms clock, 0.38+0.23/0.81/0.89+0.083 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 19 @0.148s 2%: 0.036+0.41+0.004 ms clock, 0.36+0.086/0.98/0.77+0.049 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 20 @0.348s 1%: 0.10+0.46+0.004 ms clock, 1.0+0.078/1.0/1.5+0.047 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
# go-tut/kevin
gc 1 @0.002s 2%: 0.004+0.71+0.002 ms clock, 0.041+0.55/0.21/0.004+0.028 ms cpu, 3->5->4 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 2 @0.020s 0%: 0.003+0.25+0.022 ms clock, 0.039+0.054/0.56/0.063+0.22 ms cpu, 8->9->8 MB, 9 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 3 @0.050s 0%: 0.022+0.45+0.012 ms clock, 0.22+0/0.88/0.51+0.12 ms cpu, 14->14->10 MB, 16 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 4 @0.125s 0%: 0.14+0.44+0.035 ms clock, 1.4+0.12/0.85/0.27+0.35 ms cpu, 18->18->11 MB, 21 MB goal, 0 MB stacks, 0 MB globals, 10 P
Start normalising
Input file opened
Output file opened
Channels setup
Workers started
Batched output started
Batched output started
Batched output started
Batched output started
Start processing
Lines per second:
562473.7482955287
Total lines process
500000
Completed time (ms)
888

Conclusion, we have reached the end of the experiment, we began with this program taking roughly 11 seconds to run with 1 worker and 1 writer and taking up to 1.6 GB worth of memory to run. Through the optimisations above, we have managed to achieve sub second performance with 1 worker and 1 writer while only taking up to 2 MB of memory. 

# Bug

## Concurrent Buffer Sharing Race Condition

**Problem:** Workers and writers share buffer references through channels, but workers process faster than writers. This creates a race condition where workers overwrite the underlying buffer array before writers finish reading the data, resulting in corrupted output.

**Root Cause:** In `startWorker()` line 318, `ProcessLogLine()` performs in-place modifications on a reused buffer. When this buffer reference is sent through the channel to writers, both the worker (reusing the buffer for the next line) and writer (still reading previous data) access the same underlying array concurrently.

**Current Workaround:** Allocating a new `outputBuf` on every iteration (lines 325-326) to ensure each writer gets an independent copy. This eliminates the race but defeats the memory optimization benefits achieved earlier.

**Potential Solution** Implement a bufferPool that is shared between the Worker and Writers. Workers will always pick up a clean buffer from the pool and Writers will return the buffers after writing to file. This will guarantee that no race conditions could occur. This should not increase the over memory overhead because we will
initialise the buffers but they will only grow if used by the workers. In the ideal condition, we should only expect worker + writer number of buffers actually used while
also avoiding creating temporary strings for passing around.


## Optimisation
- Created rune buffer pool. 
- Reuses buffers from the same pool, each line uses a new buffer and is released after passing the contents to the bufio writer

## Results
File: kevin
Type: cpu
Time: 2025-08-19 17:33:41 PDT
Duration: 801.88ms, Total samples = 3.27s (407.79%)
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 3.26s, 99.69% of 3.27s total
Dropped 11 nodes (cum <= 0.02s)
Showing top 10 nodes out of 53
      flat  flat%   sum%        cum   cum%
     1.99s 60.86% 60.86%      1.99s 60.86%  runtime.usleep
     0.53s 16.21% 77.06%      0.53s 16.21%  syscall.syscall
     0.40s 12.23% 89.30%      0.40s 12.23%  runtime.pthread_cond_wait
     0.14s  4.28% 93.58%      0.14s  4.28%  runtime.pthread_cond_signal
     0.06s  1.83% 95.41%      0.06s  1.83%  runtime.mapaccess2_fast32
     0.04s  1.22% 96.64%      0.12s  3.67%  main.Mask
     0.04s  1.22% 97.86%      0.04s  1.22%  main.RemoveAlphanumeric
     0.02s  0.61% 98.47%      0.60s 18.35%  main.(*LogNormaliser).Start
     0.02s  0.61% 99.08%      0.02s  0.61%  main.Compress (inline)
     0.02s  0.61% 99.69%      0.02s  0.61%  unicode/utf8.DecodeRune

➜  golang-sd git:(master) ✗ GODEBUG=gctrace=1 go run .
gc 1 @0.004s 2%: 0.022+1.5+0.061 ms clock, 0.22+0.49/0.60/0+0.61 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 2 @0.008s 2%: 0.012+0.30+0.006 ms clock, 0.12+0.013/0.67/0.68+0.069 ms cpu, 3->3->0 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 3 @0.010s 3%: 0.075+0.65+0.075 ms clock, 0.75+0.35/0.78/0+0.75 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 4 @0.012s 5%: 0.13+0.49+0.010 ms clock, 1.3+0.50/1.0/0+0.10 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 5 @0.019s 7%: 0.77+0.90+0.033 ms clock, 7.7+0.31/1.4/0+0.33 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 6 @0.023s 7%: 0.052+0.47+0.004 ms clock, 0.52+0.12/1.0/0.63+0.049 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 7 @0.024s 7%: 0.034+0.48+0.053 ms clock, 0.34+0.37/0.96/0+0.53 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 8 @0.026s 7%: 0.046+0.50+0.010 ms clock, 0.46+0.044/1.2/0.52+0.10 ms cpu, 4->4->2 MB, 5 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 9 @0.028s 7%: 0.035+0.42+0.036 ms clock, 0.35+0.018/0.86/1.1+0.36 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 10 @0.030s 7%: 0.024+0.37+0.004 ms clock, 0.24+0.052/0.86/1.1+0.049 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 11 @0.032s 7%: 0.035+0.36+0.005 ms clock, 0.35+0.016/0.95/1.0+0.052 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 12 @0.033s 7%: 0.026+0.51+0.012 ms clock, 0.26+0.041/0.99/1.1+0.12 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 13 @0.035s 7%: 0.054+0.80+0.057 ms clock, 0.54+0.10/1.5/0.014+0.57 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 14 @0.036s 7%: 0.050+0.52+0.009 ms clock, 0.50+0.040/1.1/0.13+0.097 ms cpu, 4->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 15 @0.038s 7%: 0.034+0.51+0.031 ms clock, 0.34+0.10/1.1/0.60+0.31 ms cpu, 3->4->2 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 16 @0.040s 7%: 0.045+0.35+0.003 ms clock, 0.45+0.037/0.77/0.69+0.030 ms cpu, 3->4->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P
gc 17 @0.045s 7%: 0.12+0.41+0.005 ms clock, 1.2+0/1.0/1.1+0.051 ms cpu, 3->3->1 MB, 4 MB goal, 0 MB stacks, 0 MB globals, 10 P

➜  golang-sd git:(master) ✗ go tool pprof mem.prof       
File: kevin
Type: alloc_space
Time: 2025-08-19 17:30:38 PDT
Entering interactive mode (type "help" for commands, "o" for options)
(pprof) top10
Showing nodes accounting for 2586.49kB, 100% of 2586.49kB total
Showing top 10 nodes out of 21
      flat  flat%   sum%        cum   cum%
    1026kB 39.67% 39.67%     1026kB 39.67%  runtime.allocm
  532.26kB 20.58% 60.25%   532.26kB 20.58%  main.NewRunePool (inline)
  516.01kB 19.95% 80.20%   516.01kB 19.95%  bufio.(*Scanner).Scan
  512.22kB 19.80%   100%   512.22kB 19.80%  runtime.malg
         0     0%   100%   516.01kB 19.95%  main.(*LogNormaliser).Start
         0     0%   100%   532.26kB 20.58%  main.NewLogNormaliser
         0     0%   100%  1048.27kB 40.53%  main.main
         0     0%   100%  1048.27kB 40.53%  runtime.main
         0     0%   100%      513kB 19.83%  runtime.mcall
         0     0%   100%      513kB 19.83%  runtime.mstart


It seems from the metrics that we're not encountering any GC cycles during the execution phase of the program. The GC cycles seen above
seem to be from the go compiler. We noticed a slight increase in performance with the program running around 800ms lesser than 1s previously.
Although the memory usage increased to 2.5MB, the space complexity should be constant now. Furthermore, we still fixed the bug of writers
overwriting the worker's buffers. The new design also makes the system safe as it is impossible for the same buffer to be written over. 

(pprof) peek runtime.usleep
Showing nodes accounting for 3.27s, 100% of 3.27s total
----------------------------------------------------------+-------------
      flat  flat%   sum%        cum   cum%   calls calls% + context              
----------------------------------------------------------+-------------
                                             1.91s 95.98% |   runtime.runqgrab
                                             0.08s  4.02% |   runtime.osyield
     1.99s 60.86% 60.86%      1.99s 60.86%                | runtime.usleep
----------------------------------------------------------+-------------

Considering that we know that the GC is not running during the execution phase, this should be due to waiting go routines.