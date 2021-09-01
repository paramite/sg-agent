#!/usr/bin/python3

"""Script for validating content of log file produced by print plugin"""

import datetime
import json
import socket
import sys


expected = [
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-agent-scheduler' % socket.gethostname(),
        'Severity': 2,
        "Message": "",
        'Labels': {
            'task': {
                'Name': 'test1',
                'Command': "echo 'test1'",
                'Interval': '1s',
                'Timeout': 0,
                'MuteOn': None,
                'Retries': 0,
                'CoolDown': 0,
                'Type': 'internal'
            }
        },
        'Annotations': None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-agent-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'action': 'scheduled',
            'name': 'test1',
            'command': "echo 'test1'",
            'type': 'internal'
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-agent-scheduler' %  socket.gethostname(),
        'Severity': 2,
        "Message": "",
        'Labels': {
            'task': {
                'Name': 'test2',
                'Command': "echo 'test2'",
                'Interval': '2s',
                'Timeout': 0,
                'MuteOn': None,
                'Retries': 0,
                'CoolDown': 0,
                'Type': 'internal'
            }
        },
        'Annotations': None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-agent-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'action': 'scheduled',
            'name': 'test2',
            'command': "echo 'test2'",
            'type': 'internal'
        },
        "Annotations": None
    }
]

def validate_event(evt_str):
    evt = json.loads(evt_str)
    if evt in expected:
        expected.remove(evt)

def main(log_path):
    with open(log_path) as log:
        buffer = ""
        for line in log:
            sline = line.strip()
            if sline == "{":
                buffer = sline
            else:
                buffer += sline
            if not (line.startswith(" ") or line.startswith("\t")) and sline == "}":
                validate_event(buffer)

    if expected:
        print("Did not find all expected events in log. Missing events are following:\n%s" % expected)
        sys.exit(1)
    else:
        print("All expected events were found.")


if __name__ == "__main__":
    main(sys.argv[1])
