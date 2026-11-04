"""Developer ID/notarization gates; credentials stay in the user's Keychain."""
import json
from pathlib import Path
import plistlib
import re
import subprocess
import uuid


def run(argv, timeout=120):
    return subprocess.run([str(arg) for arg in argv], check=True, text=True,
                          capture_output=True, timeout=timeout)


def developer_identity(identity, keychain=None):
    command = ['/usr/bin/security', 'find-identity', '-v', '-p', 'codesigning']
    if keychain:
        command.append(keychain)
    entries = re.findall(r'\b([0-9A-Fa-f]{40}) "([^"]+)"', run(command).stdout)
    matches = [(fingerprint.upper(), name) for fingerprint, name in entries
               if identity.upper() == fingerprint.upper() or identity == name]
    if len(matches) != 1 or not matches[0][1].startswith('Developer ID Application: '):
        raise ValueError('notarization requires an installed, valid Developer ID Application '
                         'identity (exact name or fingerprint); Apple Development is not sufficient')
    return matches[0][0]


def signature(path):
    run(['/usr/bin/codesign', '--verify', '--strict', path])
    result = run(['/usr/bin/codesign', '--display', '--verbose=4', path])
    details = result.stdout+'\n'+result.stderr
    authority = re.search(r'^Authority=(Developer ID Application: .+)$', details, re.M)
    team = re.search(r'^TeamIdentifier=([A-Z0-9]{10})$', details, re.M)
    flags = re.search(r'^CodeDirectory .*\bflags=0x([0-9a-fA-F]+)', details, re.M)
    if not authority or not team or not flags or not int(flags[1], 16) & 0x10000:
        raise ValueError('distribution executable must have Developer ID and hardened runtime')
    if not re.search(r'^Timestamp=.+$', details, re.M):
        raise ValueError('distribution executable lacks a secure signing timestamp')
    entitlements = run(['/usr/bin/codesign', '--display', '--entitlements', '-', '--xml', path]).stdout
    if entitlements and plistlib.loads(entitlements.encode()).get('com.apple.security.get-task-allow'):
        raise ValueError('distribution executable has a debugging entitlement')
    return dict(authority=authority[1], team=team[1], hardened_runtime=True, secure_timestamp=True)


def verify_distribution(app):
    app = Path(app)
    info = plistlib.loads((app/'Contents/Info.plist').read_bytes())
    if info.get('CFBundleIdentifier') != 'com.dshills.elab.web' or 'ELabWebTestDataDirectory' in info:
        raise ValueError('distribution checks require the shipping app, not a test fixture')
    signatures = {relative: signature(app/relative) for relative in
                  ['Contents/MacOS/ELabWebLauncher', 'Contents/Resources/elab-web']}
    if len({entry['team'] for entry in signatures.values()}) != 1:
        raise ValueError('launcher and backend signing teams differ')
    run(['/usr/bin/codesign', '--verify', '--deep', '--strict', app])
    run(['xcrun', 'stapler', 'validate', app])
    run(['/usr/sbin/spctl', '--assess', '--type', 'execute', '--verbose=4', app])
    policy_tool = Path('/usr/bin/syspolicy_check')
    if policy_tool.exists():
        run([policy_tool, 'distribution', app])
    return dict(signatures=signatures, stapled_ticket_valid=True, gatekeeper_accepted=True,
                system_policy_preflight=policy_tool.exists())


def notarize(app, directory, profile, keychain=None):
    submission = Path(directory)/'notary-submission.zip'
    run(['/usr/bin/ditto', '-c', '-k', '--sequesterRsrc', '--keepParent', app, submission])
    command = ['xcrun', 'notarytool', 'submit', submission, '--keychain-profile', profile,
               '--wait', '--timeout', '20m', '--output-format', 'json', '--no-progress']
    if keychain:
        command.extend(['--keychain', keychain])
    reply = json.loads(run(command, timeout=1260).stdout)
    # Never treat a pending request, tool exit alone or missing ID as acceptance.
    if reply.get('status') != 'Accepted':
        raise ValueError('Apple notarization did not accept the app: '+str(reply.get('status')))
    try:
        identifier = str(uuid.UUID(reply['id']))
    except (KeyError, TypeError, ValueError) as error:
        raise ValueError('Apple acceptance reply lacks a valid submission ID') from error
    run(['xcrun', 'stapler', 'staple', app])
    result = verify_distribution(app)
    return dict(submission_id=identifier, status='Accepted', **result)
