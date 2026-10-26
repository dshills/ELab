"""Exclusive publication: atomic hardlink or non-atomic copy fallback.

On filesystems without hardlinks the final path is incomplete until this function
returns successfully. A process crash can leave a partial path; the caller must
not consume it without the successful producer report and checksum verification.
"""
import errno
import os
from pathlib import Path
import shutil


def publish_exclusive(source, output):
    try:
        os.link(source, output)
        return
    except OSError as error:
        unsupported = {errno.EPERM, errno.EXDEV, errno.ENOSYS, errno.ENOTSUP, errno.EOPNOTSUPP}
        if error.errno not in unsupported:
            raise
    owned = None
    descriptor = None
    try:
        descriptor = os.open(output, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
        owned = os.fstat(descriptor)
        destination = os.fdopen(descriptor, "wb")
        descriptor = None  # fdopen now owns closing the descriptor.
        with destination:
            with source.open("rb") as original:
                shutil.copyfileobj(original, destination)
            destination.flush()
            os.fsync(destination.fileno())
            if hasattr(os, "fchmod"):
                os.fchmod(destination.fileno(), source.stat().st_mode & 0o777)
    except BaseException:
        # Cleanup only the file we created, never a preexisting or replaced path.
        if owned is not None:
            try:
                current = output.lstat()
                if (current.st_dev, current.st_ino) == (owned.st_dev, owned.st_ino):
                    output.unlink()
            except FileNotFoundError:
                pass
        raise

    finally:
        if descriptor is not None:
            os.close(descriptor)
