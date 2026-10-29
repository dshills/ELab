"""Verify archived study provenance without qualifying the current application."""
import hashlib
import json
import re

from wiring_replay import LEDGER, git_blob, git_files, runtime_input


def validate_origin(root, checkpoint, identity, receipt_bytes):
    """No executable is run and no current source/binary match is asserted."""
    receipt_sha = hashlib.sha256(receipt_bytes).hexdigest()
    if receipt_sha != checkpoint['review_receipt_sha256'] or identity['receipt_sha256'] != receipt_sha:
        raise ValueError('archived review receipt identity differs')
    receipt = json.loads(receipt_bytes)
    source = receipt['source_commit']
    if not re.fullmatch(r'[0-9a-f]{40}', source) or receipt.get('clean_build_vcs') is not True:
        raise ValueError('archived review requires a clean source identity')
    if identity['source_commit'] != source or identity['release'] != checkpoint['build'] or identity['release'] != receipt['version']+' /'+receipt['build']:
        raise ValueError('archived release identity differs')
    for kind in ['agent', 'native']:
        if receipt[kind+'_sha256'] != checkpoint[kind+'_sha256']:
            raise ValueError('archived '+kind+' pin differs')
    if identity['agent_sha256'] != checkpoint['agent_sha256']:
        raise ValueError('recorded agent identity differs')
    ledger_bytes = git_blob(root, source, LEDGER)
    ledger = json.loads(ledger_bytes)
    if hashlib.sha256(ledger_bytes).hexdigest() != identity['ledger_sha256']:
        raise ValueError('archived ledger identity differs')
    if len(ledger['source_sha256']) != identity['production_source_files_verified']:
        raise ValueError('archived source count differs')
    for name, expected in ledger['source_sha256'].items():
        if hashlib.sha256(git_blob(root, source, name)).hexdigest() != expected:
            raise ValueError('archived source hash differs: '+name)
    if sum(runtime_input(name) for name in git_files(root, source)) != identity['runtime_inputs_verified']:
        raise ValueError('archived runtime input count differs')
    if json.loads(git_blob(root, source, 'third_party/vellum/sdk.lock.json')) != receipt['sdk']:
        raise ValueError('archived SDK identity differs')
    if identity['manifest_file_entries_verified'] != receipt['manifest_file_entries']:
        raise ValueError('archived manifest count differs')
    return identity
