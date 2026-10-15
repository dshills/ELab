"""Keep full discovery witnesses separate from compact workflow reports."""
import hashlib
import json
from pathlib import Path


def discovery_summary(response, destination):
    """Write a new complete response and return contract fields plus digests."""
    registry = response['schemas']
    if not isinstance(registry, dict) or not registry:
        raise ValueError('expected a nonempty discovery schema registry')
    encode = lambda value: (json.dumps(value, sort_keys=True, indent=2, allow_nan=False)+'\n').encode()
    full = encode(response)
    path = Path(destination)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('xb') as artifact:
        artifact.write(full)
    result = {key: value for key, value in response.items() if key != 'schemas'}
    result.update(schema_registry_count=len(registry),
                  schema_registry_sha256=hashlib.sha256(encode(registry)).hexdigest(),
                  full_response_sha256=hashlib.sha256(full).hexdigest())
    return result
