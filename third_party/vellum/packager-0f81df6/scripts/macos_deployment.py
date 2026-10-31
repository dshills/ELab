"""Read executable deployment declarations; never select or lower a deployment target."""
import re
from child_process import check_child_output
from sdk_environment import child_environment


def version(value):
    if not isinstance(value,str) or not re.fullmatch(r'\d{1,3}(?:\.\d{1,3}){0,2}',value):
        raise ValueError('invalid macOS deployment version')
    parts=[int(x) for x in value.split('.')]
    return tuple(parts+[0]*(3-len(parts)))


def parse(output):
    deployments=[]
    for block in re.split(r'(?m)^Load command \d+\s*$',output)[1:]:
        def field(key):
            values=re.findall(r'^\s*'+key+r'\s+(\S+)\s*$',block,re.M)
            if len(values)!=1:raise ValueError('missing/duplicate deployment field: '+key)
            return values[0]
        cmd=field('cmd')
        if cmd not in {'LC_BUILD_VERSION','LC_VERSION_MIN_MACOSX'}:continue
        if cmd=='LC_BUILD_VERSION' and field('platform') not in {'1','macos','MACOS'}:
            raise ValueError('deployment platform is not macOS')
        minimum=field('minos' if cmd=='LC_BUILD_VERSION' else 'version');sdk=field('sdk')
        if version(minimum)>version(sdk):raise ValueError('deployment minimum exceeds SDK')
        deployments.append(dict(minimum=minimum,sdk=sdk))
    if len(deployments)!=1:raise ValueError('missing or ambiguous macOS deployment metadata')
    return deployments[0]


def inspect(binary):
    def tool(*args):return check_child_output(list(args),env=child_environment(),text=True,timeout=20)
    architectures=tool('lipo','-archs',str(binary)).split()
    if (not 1<=len(architectures)<=8 or len(set(architectures))!=len(architectures)
            or any(not re.fullmatch(r'[A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)*',a) for a in architectures)):
        raise ValueError('invalid Mach-O architecture list')
    records={a:parse(tool('otool','-arch',a,'-l',str(binary))) for a in architectures}
    # A universal bundle must not advertise a floor lower than any shipped slice.
    minimum=max((r['minimum'] for r in records.values()),key=version)
    return dict(minimum=minimum,architectures=records)


def verify(info,declared,observed):
    if declared is None:
        if 'LSMinimumSystemVersion' in info:raise ValueError('unbound minimum OS metadata')
        return  # Legacy bundles explicitly have no recorded minimum field.
    if declared!=observed or info.get('LSMinimumSystemVersion')!=observed['minimum']:
        raise ValueError('minimum OS metadata differs from executable deployment')
