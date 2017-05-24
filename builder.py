import datetime
import re
import sys
import subprocess
import math
import json

CONSTANTS_PATH = 'constants/constants.go'
STABLE_PACUR_PATH = '../pritunl-pacur'
TEST_PACUR_PATH = '../pritunl-pacur-test'
BUILD_KEYS_PATH = 'build_keys.json'
BUILD_TARGETS = ('pritunl-link',)

cur_date = datetime.datetime.utcnow()

with open(BUILD_KEYS_PATH, 'r') as build_keys_file:
    build_keys = json.loads(build_keys_file.read().strip())
    mirror_url = build_keys['mirror_url']
    test_mirror_url = build_keys['test_mirror_url']

def get_ver(version):
    day_num = (cur_date - datetime.datetime(2015, 11, 24)).days
    min_num = int(math.floor(((cur_date.hour * 60) + cur_date.minute) / 14.4))
    ver = re.findall(r'\d+', version)
    ver_str = '.'.join((ver[0], ver[1], str(day_num), str(min_num)))
    ver_str += ''.join(re.findall('[a-z]+', version))

    return ver_str

def get_int_ver(version):
    ver = re.findall(r'\d+', version)

    if 'snapshot' in version:
        pass
    elif 'alpha' in version:
        ver[-1] = str(int(ver[-1]) + 1000)
    elif 'beta' in version:
        ver[-1] = str(int(ver[-1]) + 2000)
    elif 'rc' in version:
        ver[-1] = str(int(ver[-1]) + 3000)
    else:
        ver[-1] = str(int(ver[-1]) + 4000)

    return int(''.join([x.zfill(4) for x in ver]))

cmd = sys.argv[1]

if cmd == 'set-version':
    new_version = get_ver(sys.argv[2])

    with open(CONSTANTS_PATH, 'r') as constants_file:
        constants_data = constants_file.read()

    with open(CONSTANTS_PATH, 'w') as constants_file:
        constants_file.write(re.sub(
            '(= ".*?")',
            '= "%s"' % new_version,
            constants_data,
            count=1,
        ))

    subprocess.check_call(['git', 'reset', 'HEAD', '.'])
    subprocess.check_call(['git', 'add', CONSTANTS_PATH])
    subprocess.check_call(['git', 'commit', '-S', '-m', 'Create new release'])
    subprocess.check_call(['git', 'push'])

elif cmd == 'build':
    for build_target in BUILD_TARGETS:
        subprocess.check_call(
            ['sudo', 'pacur', 'project', 'build', build_target],
            cwd=STABLE_PACUR_PATH,
        )

elif cmd == 'build-test':
    for build_target in BUILD_TARGETS:
        subprocess.check_call(
            ['sudo', 'pacur', 'project', 'build', build_target],
            cwd=TEST_PACUR_PATH,
        )

elif cmd == 'upload':
    subprocess.check_call(
        ['sudo', 'pacur', 'project', 'repo'],
        cwd=STABLE_PACUR_PATH,
    )

    for mir_url in mirror_url:
        subprocess.check_call([
            'rsync',
            '--human-readable',
            '--archive',
            '--progress',
            '--delete',
            '--acls',
            'mirror/',
            mir_url,
        ], cwd=STABLE_PACUR_PATH)

elif cmd == 'upload-test':
    subprocess.check_call(
        ['sudo', 'pacur', 'project', 'repo'],
        cwd=TEST_PACUR_PATH,
    )

    for mir_url in test_mirror_url:
        subprocess.check_call([
            'rsync',
            '--human-readable',
            '--archive',
            '--progress',
            '--delete',
            '--acls',
            'mirror/',
            mir_url,
        ], cwd=TEST_PACUR_PATH)
