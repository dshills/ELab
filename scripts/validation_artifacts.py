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


def discovery_bundle(responses, destination):
    """Save multiple complete filtered responses with one identical registry.

    Reconstruct each response by adding ``schemas`` to its ``queries`` entry.
    Per-response hashes use the same canonical encoding as discovery_summary.
    """
    if not responses:
        raise ValueError('expected at least one filtered response')
    encode = lambda value: (json.dumps(value, sort_keys=True, indent=2, allow_nan=False)+'\n').encode()
    registry = next(iter(responses.values())).get('schemas')
    if not isinstance(registry, dict) or not registry:
        raise ValueError('expected a nonempty discovery schema registry')
    queries, digests = {}, {}
    for command, response in responses.items():
        if response.get('schemas') != registry:
            raise ValueError('filtered responses have different schema registries')
        queries[command] = {key: value for key, value in response.items() if key != 'schemas'}
        reconstructed = dict(queries[command], schemas=registry)
        if reconstructed != response:
            raise ValueError('response reconstruction failed')
        digests[command] = hashlib.sha256(encode(response)).hexdigest()
    bundle = dict(format='elab.discovery-bundle', version=1, queries=queries, schemas=registry)
    path = Path(destination)
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open('xb') as artifact:
        artifact.write(encode(bundle))
    return dict(bundle_sha256=hashlib.sha256(encode(bundle)).hexdigest(),
                schema_registry_count=len(registry),
                schema_registry_sha256=hashlib.sha256(encode(registry)).hexdigest(),
                full_response_sha256=digests)
