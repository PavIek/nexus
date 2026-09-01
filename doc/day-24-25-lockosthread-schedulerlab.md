# Day 24-25: runtime.LockOSThread Scheduler Lab

## Experiment Goal

This experiment observes how Go schedules goroutines onto OS threads, and how
`runtime.LockOSThread()` changes that behavior.

The main question:

```text
Can the same goroutine move between different OS threads?
```

The answer from this experiment:

```text
Without LockOSThread:
the main goroutine can migrate between OS threads.

With LockOSThread:
the main goroutine stays on the OS thread where it was locked.
```

## Commands

Build the experiment binary:

```bash
go build -o /tmp/schedulerlab ./cmd/schedulerlab
```

Run without locking the main goroutine to an OS thread:

```bash
GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8
```

Run with the main goroutine locked to its current OS thread:

```bash
LOCK_OS_THREAD=1 GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8
```

## Case 1: Without LockOSThread

Start of the run:

```text
pid=872037 main_tid=872037 locked_at_init=false gomaxprocs=4 workers=8 duration=6s
```

Later output:

```text
main_tid=872039 goroutines=9
main_tid=872037 goroutines=9
done main_tid=872041
```

Analysis:

The main goroutine started on OS thread `872037`, later ran on `872039`, then
finished on `872041`.

This proves that the main goroutine was not bound to one OS thread. The Go
runtime was free to resume it on different `M`s.

Worker goroutines also moved between OS threads:

```text
worker=07 tid=872037
worker=07 tid=872040
worker=07 tid=872041
worker=07 tid=872039
```

This is normal Go scheduler behavior. Goroutines are user-space tasks managed by
the Go runtime. They are not, by default, permanently attached to a specific OS
thread.

Relevant `schedtrace` sample:

```text
SCHED 1005ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=2 [ 1 1 0 0 ] schedticks=[ 92 90 91 85 ]
```

Meaning:

- `gomaxprocs=4`: there are 4 `P`s available to run Go code.
- `idleprocs=0`: all 4 `P`s are busy.
- `threads=7`: the runtime created 7 OS threads.
- `runqueue=2`: some goroutines are waiting in the global run queue.
- `[ 1 1 0 0 ]`: per-`P` local run queue lengths.

## Case 2: With LockOSThread

Start of the run:

```text
pid=872494 main_tid=872494 locked_at_init=true gomaxprocs=4 workers=8 duration=6s
```

Later output:

```text
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
done main_tid=872494
```

Analysis:

The main goroutine started on OS thread `872494` and finished on the same OS
thread.

This proves that `runtime.LockOSThread()` worked. The current goroutine was
bound to the current OS thread.

Important detail:

```text
LockOSThread only locks the goroutine that calls it.
```

Other worker goroutines were still free to move between OS threads:

```text
worker=07 tid=872496
worker=07 tid=872500
worker=07 tid=872498
worker=07 tid=872501
```

Relevant `schedtrace` sample:

```text
SCHED 1009ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=3 [ 0 0 1 1 ] schedticks=[ 81 91 83 89 ]
```

Compared with the unlocked run, `threads` increased from `7` to `8`.

That is expected. Once the main goroutine owns a specific OS thread, the runtime
may create additional OS threads so that other goroutines can continue running
normally.

## Why SCHED 0ms Shows gomaxprocs=28

Both runs start with a line like this:

```text
SCHED 0ms: gomaxprocs=28 ...
```

This line appears before `main()` calls:

```go
runtime.GOMAXPROCS(4)
```

So the first scheduler snapshot still shows the default value, which is based on
the machine's CPU count.

The later snapshots show the actual experiment setting:

```text
gomaxprocs=4
```

## Key Concepts

### G, M, P

Go's scheduler is usually explained using the GMP model:

- `G`: goroutine.
- `M`: machine, which means an OS thread.
- `P`: processor resource required to execute Go code.

`GOMAXPROCS=4` means at most 4 `P`s can execute Go code at the same time.

It does not mean the runtime will only create 4 OS threads.

### Goroutine Migration

By default, a goroutine can resume execution on different OS threads over time.

This gives the runtime flexibility for:

- load balancing
- work stealing
- handling blocking syscalls
- using available CPU resources efficiently

### Thread Affinity

`runtime.LockOSThread()` creates thread affinity for the current goroutine.

After locking:

```text
current G -> current M
```

The current goroutine must keep running on that OS thread.

This is useful when external systems require same-thread execution, such as:

- GUI main loops
- OpenGL or other graphics contexts
- C/C++ libraries using thread-local storage
- OS APIs with per-thread state
- special runtime or scheduler experiments

It is usually not needed for normal Go servers, worker pools, network handling,
or CPU tasks.

## Final Conclusion

The experiment confirms:

```text
P controls Go-level parallelism.
M is an OS thread.
G is a goroutine.
G can normally migrate between M.
LockOSThread binds the current G to the current M.
```

For the `nexus` project, `LockOSThread` is not needed for the TCP server,
ringbuffer, or sharded map implementation. This experiment is mainly for
understanding Go runtime scheduling behavior.

## Raw Output

### Without LockOSThread

```text
09:07:21 euclid@Greece nexus +/-|main *|-> GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8
SCHED 0ms: gomaxprocs=28 idleprocs=25 threads=5 spinningthreads=1 needspinning=0 idlethreads=0 runqueue=0 [ 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 ] schedticks=[ 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2 ]
pid=872037 main_tid=872037 locked_at_init=false gomaxprocs=4 workers=8 duration=6s
worker=01 tid=872041
worker=00 tid=872040
worker=04 tid=872039
worker=05 tid=872037
worker=07 tid=872037
worker=01 tid=872039
worker=04 tid=872042
worker=05 tid=872043
worker=03 tid=872037
worker=02 tid=872037
worker=00 tid=872042
worker=06 tid=872037
worker=05 tid=872040
worker=07 tid=872040
worker=04 tid=872043
worker=02 tid=872040
worker=00 tid=872041
worker=06 tid=872039
worker=05 tid=872043
worker=03 tid=872041
worker=04 tid=872041
worker=01 tid=872041
worker=00 tid=872042
worker=07 tid=872041
worker=02 tid=872042
worker=01 tid=872040
worker=00 tid=872041
worker=05 tid=872041
worker=06 tid=872043
worker=02 tid=872039
worker=03 tid=872040
worker=00 tid=872042
worker=04 tid=872043
worker=07 tid=872043
worker=03 tid=872039
worker=04 tid=872040
worker=02 tid=872043
worker=06 tid=872039
worker=00 tid=872040
worker=03 tid=872040
worker=04 tid=872042
worker=01 tid=872039
worker=07 tid=872039
worker=02 tid=872041
worker=06 tid=872041
worker=03 tid=872039
worker=04 tid=872040
worker=00 tid=872042
worker=03 tid=872041
main_tid=872039 goroutines=9
worker=01 tid=872040
worker=05 tid=872043
worker=06 tid=872039
worker=03 tid=872040
worker=01 tid=872039
worker=05 tid=872042
worker=06 tid=872040
worker=07 tid=872041
worker=01 tid=872040
worker=02 tid=872042
worker=05 tid=872039
worker=02 tid=872037
worker=06 tid=872041
worker=07 tid=872039
worker=07 tid=872037
main_tid=872037 goroutines=9
SCHED 1005ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=2 [ 1 1 0 0 ] schedticks=[ 92 90 91 85 ]
main_tid=872037 goroutines=9
main_tid=872037 goroutines=9
SCHED 2011ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=2 [ 0 1 0 1 ] schedticks=[ 171 166 184 166 ]
main_tid=872037 goroutines=9
main_tid=872037 goroutines=9
SCHED 3018ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=2 [ 1 1 0 0 ] schedticks=[ 254 234 278 247 ]
main_tid=872037 goroutines=9
main_tid=872037 goroutines=9
SCHED 4026ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=3 [ 0 1 0 0 ] schedticks=[ 339 303 348 355 ]
main_tid=872037 goroutines=9
main_tid=872037 goroutines=9
SCHED 5036ms: gomaxprocs=4 idleprocs=0 threads=7 spinningthreads=0 needspinning=1 idlethreads=2 runqueue=3 [ 0 1 0 0 ] schedticks=[ 418 392 425 441 ]
main_tid=872037 goroutines=9
main_tid=872037 goroutines=9
done main_tid=872041
```

### With LockOSThread

```text
09:07:38 euclid@Greece nexus +/-|main *|-> LOCK_OS_THREAD=1 GODEBUG=schedtrace=1000 /tmp/schedulerlab -duration=6s -procs=4 -workers=8
SCHED 0ms: gomaxprocs=28 idleprocs=25 threads=5 spinningthreads=1 needspinning=0 idlethreads=0 runqueue=0 [ 1 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 ] schedticks=[ 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 0 2 ]
pid=872494 main_tid=872494 locked_at_init=true gomaxprocs=4 workers=8 duration=6s
worker=07 tid=872496
worker=02 tid=872497
worker=01 tid=872500
worker=00 tid=872498
worker=03 tid=872496
worker=04 tid=872496
worker=05 tid=872501
worker=03 tid=872500
worker=06 tid=872501
worker=07 tid=872500
worker=02 tid=872498
worker=01 tid=872497
worker=04 tid=872498
worker=05 tid=872500
worker=02 tid=872496
worker=03 tid=872496
worker=05 tid=872498
worker=07 tid=872498
worker=00 tid=872496
worker=04 tid=872500
worker=06 tid=872500
worker=02 tid=872500
worker=01 tid=872496
worker=06 tid=872501
worker=03 tid=872498
worker=07 tid=872497
worker=06 tid=872497
worker=01 tid=872500
worker=04 tid=872498
worker=07 tid=872501
worker=00 tid=872498
worker=02 tid=872501
worker=07 tid=872496
worker=05 tid=872501
worker=00 tid=872501
worker=02 tid=872496
worker=06 tid=872498
worker=03 tid=872501
worker=05 tid=872496
worker=04 tid=872501
worker=07 tid=872501
worker=03 tid=872500
worker=00 tid=872496
worker=06 tid=872500
main_tid=872494 goroutines=9
worker=02 tid=872501
worker=01 tid=872501
worker=04 tid=872496
worker=05 tid=872497
worker=02 tid=872498
worker=03 tid=872496
worker=04 tid=872501
worker=01 tid=872498
worker=05 tid=872500
worker=07 tid=872498
worker=03 tid=872498
worker=01 tid=872500
worker=05 tid=872501
worker=06 tid=872496
worker=00 tid=872497
worker=04 tid=872496
worker=01 tid=872497
worker=00 tid=872500
worker=00 tid=872501
worker=06 tid=872501
SCHED 1009ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=3 [ 0 0 1 1 ] schedticks=[ 81 91 83 89 ]
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
SCHED 2016ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=5 [ 0 0 0 0 ] schedticks=[ 161 167 162 183 ]
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
SCHED 3022ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=2 [ 0 1 0 1 ] schedticks=[ 247 239 240 276 ]
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
SCHED 4029ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=2 [ 1 0 1 0 ] schedticks=[ 324 319 332 352 ]
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
SCHED 5035ms: gomaxprocs=4 idleprocs=0 threads=8 spinningthreads=0 needspinning=1 idlethreads=1 runqueue=2 [ 0 1 1 0 ] schedticks=[ 401 401 411 441 ]
main_tid=872494 goroutines=9
main_tid=872494 goroutines=9
done main_tid=872494
```
