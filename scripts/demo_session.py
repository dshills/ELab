"""Admission for destructive demos; general-purpose clients remain unrestricted."""


class DemoSessionRefused(RuntimeError):
    """The supplied session is not a fresh disposable demonstration target."""


def require_fresh_demo_session(snapshot):
    """Read-only check; callers must independently own the supplied process.

    A blank drawing alone is insufficient: saved empty files, cleared sessions,
    undo history and retained captures may still be valuable user work. This is
    an admission check, not an ownership credential or an exclusive editing lease.
    """
    reasons = []
    if not isinstance(snapshot, dict):
        reasons.append('incompatible snapshot')
    else:
        if snapshot.get('closed') is not False:
            reasons.append('closed or unknown session')
        if snapshot.get('components') != [] or snapshot.get('wires') != []:
            reasons.append('existing drawing')
        document = snapshot.get('document')
        if not isinstance(document, dict):
            reasons.append('missing document identity')
        elif (type(document.get('generation')) is not int or document['generation'] != 1
              or document.get('path') != '' or document.get('dirty') is not False
              or document.get('schematic_dirty') is not False
              or document.get('preferences_dirty') is not False
              or document.get('recordings_dirty') is not False
              or type(document.get('recording_count')) is not int
              or document['recording_count'] != 0):
            reasons.append('saved, edited, recorded or replaced document')
        if snapshot.get('undo_entry') or snapshot.get('redo_entry'):
            reasons.append('existing history')
        if snapshot.get('operations') != []:
            reasons.append('retained operations')
        project_work = snapshot.get('project_work')
        startup_setup = (isinstance(project_work, dict)
            and project_work.get('command') in ('recovery.configure', 'recovery.configure.start')
            and project_work.get('status') == 'completed'
            and project_work.get('phase') == 'completed'
            and project_work.get('recovery_configuration_applied') is True
            and type(project_work.get('document_generation')) is int
            and project_work['document_generation'] == 1
            and type(project_work.get('result_document_generation')) is int
            and project_work['result_document_generation'] == 1
            and type(project_work.get('bytes')) is int
            and project_work['bytes'] == 0 and not project_work.get('error'))
        if ((project_work and not startup_setup)
                or any(snapshot.get(key) for key in ('step_work', 'input_work',
                    'actuation_work', 'print_preview', 'shutdown', 'last_operation'))):
            reasons.append('existing work or preview')
    if reasons:
        # Do not include private paths, IDs, circuit values or endpoint details.
        raise DemoSessionRefused('demo requires a fresh blank owned session: '+', '.join(reasons))


def connect_demo_client(factory, endpoint=None, **kwargs):
    """Admit before returning a client to a demo's destructive try/finally.

    On refusal, close only the forwarding client. Never send app.close or any
    mutation to the supplied host. No admission restriction is added to Client
    itself, so separate inspectors and observers can still attach to user work.
    """
    client = factory(endpoint, **kwargs)
    if endpoint:
        try:
            require_fresh_demo_session(client.ask('project.inspect'))
        except BaseException as error:
            try:
                client.close()
            except Exception as cleanup_error:
                raise error from cleanup_error
            raise
    return client
