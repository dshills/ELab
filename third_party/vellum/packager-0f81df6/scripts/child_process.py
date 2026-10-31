"""Bound SDK commands and clean their owned groups on timeout/interruption.

Normal completion retains ordinary subprocess semantics; this is not a sandbox.
"""
import os
import hashlib
import selectors
import signal
import subprocess
import time
import threading

CLEANUP_TIMEOUT_SECONDS = 5

class ChildCleanupError(OSError):
    """The command failed and descendant cleanup could not be confirmed."""


class ChildOutputLimitError(OSError):
    """A bounded binary log exceeded its admitted byte count."""


class ChildCancelled(SystemExit):
    """Default POSIX termination after the active SDK child has been retired."""

    def __init__(self, signum):
        self.signal_number = signum
        super().__init__(128 + signum)


class _CancellationSignals:
    """Defer cancellation until a newly created Popen can be cleaned up.

    Only the main thread's default SIGINT/SIGTERM policy is temporarily owned.
    Custom/ignored handlers and worker-thread signal policy remain the caller's.
    No signal mask or preexec callback is inherited by the child.
    """

    def __init__(self):
        self.previous = {}
        self.pending = None
        self.running = False

    def __enter__(self):
        if os.name == "nt" or threading.current_thread() is not threading.main_thread():
            return self
        try:
            for signum in (signal.SIGINT, signal.SIGTERM):
                previous = signal.getsignal(signum)
                if previous == signal.SIG_DFL or (signum == signal.SIGINT and previous is signal.default_int_handler):
                    signal.signal(signum, self.handle)
                    self.previous[signum] = previous
        except BaseException:
            self.restore()
            raise
        return self

    def handle(self, signum, frame):
        if self.pending is None:
            self.pending = signum
        # Popen must return ownership before interruption may unwind. Once an
        # error starts cleanup, further signals cannot interrupt group retirement.
        if self.running:
            self.running = False
            self.raise_pending()

    def raise_pending(self):
        if self.pending == signal.SIGINT:
            raise KeyboardInterrupt
        if self.pending is not None:
            raise ChildCancelled(self.pending)

    def start(self):
        self.running = True
        self.raise_pending()

    def cleanup(self):
        self.running = False

    def restore(self):
        self.cleanup()
        # Restoring Python's default SIGINT handler can immediately raise a
        # KeyboardInterrupt. Restore it last so other handlers are never stranded.
        for signum in reversed(self.previous):
            signal.signal(signum, self.previous[signum])
        self.previous.clear()

    def __exit__(self, exception_type, exception, traceback):
        self.restore()
        if exception_type is None:
            self.raise_pending()


def stream_logs(process, arguments, timeout, outputs, limit):
    deadline = time.monotonic() + timeout
    counts = [0, 0]
    digests = [hashlib.sha256(), hashlib.sha256()]
    with selectors.DefaultSelector() as ready:
        for index, pipe in enumerate((process.stdout, process.stderr)):
            ready.register(pipe, selectors.EVENT_READ, index)
        while ready.get_map():
            remaining = deadline - time.monotonic()
            if remaining <= 0:
                raise subprocess.TimeoutExpired(arguments, timeout)
            for key, _ in ready.select(remaining):
                chunk = os.read(key.fd, 64 * 1024)
                if not chunk:
                    ready.unregister(key.fileobj)
                    continue
                index = key.data
                accepted = min(len(chunk), limit - counts[index])
                if accepted:
                    if outputs[index].write(chunk[:accepted]) != accepted:
                        raise OSError("short binary log write")
                    digests[index].update(chunk[:accepted])
                    counts[index] += accepted
                if accepted != len(chunk):
                    raise ChildOutputLimitError("SDK child exceeded binary log byte limit")
    remaining = deadline - time.monotonic()
    if remaining <= 0:
        raise subprocess.TimeoutExpired(arguments, timeout)
    process.wait(timeout=remaining)
    return {name: {"bytes": counts[index], "sha256": digests[index].hexdigest()} for index, name in enumerate(("stdout", "stderr"))}


def terminate_owned_group(process):
    if os.name == "nt":
        # Fixed argv only; the PID belongs to the Popen instance we created.
        completed = subprocess.run(["taskkill", "/PID", str(process.pid), "/T", "/F"],
                                   stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
                                   timeout=CLEANUP_TIMEOUT_SECONDS, check=False)
        if completed.returncode != 0:
            raise ChildCleanupError("Windows taskkill could not confirm owned process-tree termination")
    else:
        try:
            os.killpg(process.pid, signal.SIGKILL)
        except ProcessLookupError:
            pass  # The entire owned group has already exited.


def run_child(arguments, *, timeout, check=False, capture_output=False, input=None, log_limit_bytes=None, **options):
    if timeout <= 0 or options.get("shell"):
        raise ValueError("SDK children require a positive deadline and argv without a shell")
    logs = None
    if log_limit_bytes is not None:
        if os.name == "nt" or type(log_limit_bytes) is not int or log_limit_bytes <= 0 or capture_output or input is not None or any(options.get(key) for key in ("text", "universal_newlines", "encoding", "errors")):
            raise ValueError("bounded logs require POSIX binary output files and a positive byte limit")
        logs = (options.get("stdout"), options.get("stderr"))
        if not all(hasattr(stream, "write") for stream in logs):
            raise ValueError("bounded logs require both binary output files")
        options["stdout"] = options["stderr"] = subprocess.PIPE
    if capture_output:
        if "stdout" in options or "stderr" in options:
            raise ValueError("capture_output conflicts with explicit stdout/stderr")
        options["stdout"] = options["stderr"] = subprocess.PIPE
    if input is not None:
        if "stdin" in options:
            raise ValueError("input conflicts with explicit stdin")
        options["stdin"] = subprocess.PIPE
    if os.name == "nt":
        options["creationflags"] = options.get("creationflags", 0) | subprocess.CREATE_NEW_PROCESS_GROUP
    else:
        options["start_new_session"] = True
    with _CancellationSignals() as cancellation:
        process = None
        communication_completed = False
        group_cleanup_attempted = False
        log_summaries = None
        try:
            process = subprocess.Popen(arguments, **options)
            try:
                cancellation.start()
                if logs is None:
                    stdout, stderr = process.communicate(input=input, timeout=timeout)
                else:
                    log_summaries = stream_logs(process, arguments, timeout, logs, log_limit_bytes)
                    stdout = stderr = None
                communication_completed = True
            except BaseException as original:
                cancellation.cleanup()
                cleanup_error = None
                try:
                    group_cleanup_attempted = True
                    terminate_owned_group(process)
                except Exception as error:
                    cleanup_error = error
                    if process.poll() is None:
                        process.kill()
                try:
                    if logs is None:
                        stdout, stderr = process.communicate(timeout=CLEANUP_TIMEOUT_SECONDS)
                    else:
                        # Do not buffer a flood during failure cleanup. These logs
                        # already contain only the admitted prefix. Closing read
                        # ends also revokes pipe writers outside the owned group.
                        process.stdout.close(); process.stderr.close()
                        process.wait(timeout=CLEANUP_TIMEOUT_SECONDS)
                        stdout = stderr = None
                except subprocess.TimeoutExpired as error:
                    raise ChildCleanupError("SDK child pipes remained open after cleanup attempt" + (": " + str(cleanup_error) if cleanup_error is not None else "")) from original
                if cleanup_error is not None:
                    raise ChildCleanupError("SDK descendant cleanup failed: " + str(cleanup_error)) from original
                if isinstance(original, subprocess.TimeoutExpired):
                    original.output, original.stderr = stdout, stderr
                raise
            completed = subprocess.CompletedProcess(arguments, process.returncode, stdout, stderr)
            if log_summaries is not None:
                completed.log_summaries = log_summaries
            if check and completed.returncode:
                raise subprocess.CalledProcessError(completed.returncode, arguments, output=stdout, stderr=stderr)
            return completed
        finally:
            cancellation.cleanup()
            if process is not None:
                # Cancellation can arrive while unwinding an unrelated error,
                # before its except block reaches group cleanup. The outer
                # finally must still retire descendants, not just the leader.
                if not communication_completed and not group_cleanup_attempted:
                    terminate_owned_group(process)
                for stream in (process.stdin, process.stdout, process.stderr):
                    if stream is not None:
                        stream.close()
                if process.poll() is None:
                    process.kill()
                    try:
                        process.wait(timeout=CLEANUP_TIMEOUT_SECONDS)
                    except subprocess.TimeoutExpired as error:
                        raise ChildCleanupError("SDK direct child remained alive after termination") from error


def check_child_output(arguments, *, timeout, **options):
    if "stdout" in options:
        raise ValueError("check_child_output owns stdout capture")
    return run_child(arguments, timeout=timeout, check=True, stdout=subprocess.PIPE, **options).stdout
