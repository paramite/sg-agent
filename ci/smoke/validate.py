#!/usr/bin/python3

"""Script for validating content of log file produced by print plugin"""

import datetime
import json
import socket
import sys


expected = [
    # scheculer events
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'instructions': {
                'Timeout': 0,
                'MuteOn': None,
                'Retries': 1,
                'CoolDown': 0
            },
            'task': {
                'Name': 'test1',
                'Command': "echo 'test1'",


                'Type': 'internal'
            }
        },
        'Annotations': None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'name': 'test1',
            'command': "echo 'test1'",
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-scheduler' %  socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'instructions': {
                'Timeout': 0,
                'MuteOn': None,
                'Retries': 4,
                'CoolDown': 1
            },
            'task': {
                'Name': 'test2',
                'Command': "echo 'test2' && exit 1"
            }
        },
        'Annotations': None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'name': 'test2',
            'command': "echo 'test2' && exit 1",
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-scheduler' %  socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'instructions': {
                'Timeout': 0,
                'Retries': 1,
                'CoolDown': 0,
                'MuteOn': [3]
            },
            'task': {
                'Name': 'test3',
                'Command': "echo 'test3' && exit 3"
            }
        },
        "Annotations": None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'name': 'test3',
            'command': "echo 'test3' && exit 3",
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'task',
        'Publisher': '%s-scheduler' %  socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'instructions': {
                'Timeout': 0,
                'Retries': 1,
                'CoolDown': 0,
                'MuteOn': None
            },
            'task': {
                'Name': 'test4',
                'Command': "echo 'test4'"
            }
        },
        'Annotations': None
    },
    {   'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Scheduled task execution request submitted for execution.',
        'Labels': {
            'name': 'test4',
            'command': "echo 'test4'",
        },
        "Annotations": None
    },
    # executor events
    {
        'Index': '',
        'Type': 'result',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'result': {
                'Task': {
                    'Name': 'test1',
                    'Command': "echo 'test1'"
                },
                'Requested': -666,
                'Requestor': '%s-scheduler' % socket.gethostname(),
                'Executor': '%s-executor' % socket.gethostname(),
                'Attempts': [
                    {
                        'Executed': -666,
                        'Duration': -666,
                        'ReturnCode': 0,
                        'StdOut': 'test1\n',
                        'StdErr': ''
                    }
                ],
                "Status": "success"
            }
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-scheduler' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 1. attempt -> RC: 0',
        'Labels': {
            "command": "echo 'test1'",
            "name": "test1"
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'result',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'result': {
                'Task': {
                    'Name': 'test2',
                    'Command': "echo 'test1'"
                },
                'Requested': -666,
                'Requestor': '%s-scheduler' % socket.gethostname(),
                'Executor': '%s-executor' % socket.gethostname(),
                'Attempts': [
                    {
                        'Executed': -666,
                        'Duration': -666,
                        'ReturnCode': 0,
                        'StdOut': 'test1\n',
                        'StdErr': ''
                    }
                ],
                "Status": "success"
            }
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 1. attempt -> RC: 0',
        'Labels': {
            "command": "echo 'test1'",
            "name": "test1"
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'result',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
        'result': {
            'Task': {
                'Name': 'test2',
                'Command': "echo 'test2' && exit 1"
            },
            'Requested': -666,
            'Requestor': '%s-scheduler' % socket.gethostname(),
            'Executor': '%s-executor' % socket.gethostname(),
            'Attempts': [
                {
                    'Executed': -666,
                    'Duration': -666,
                    'ReturnCode': 1,
                    'StdOut': 'test2\n',
                    'StdErr': ''
                },
                {
                    'Executed': -666,
                    'Duration': -666,
                    'ReturnCode': 1,
                    'StdOut': 'test2\n',
                    'StdErr': ''
                },
                {
                    'Executed': -666,
                    'Duration': -666,
                    'ReturnCode': 1,
                    'StdOut': 'test2\n',
                    'StdErr': ''
                },
                {
                    'Executed': -666,
                    'Duration': -666,
                    'ReturnCode': 1,
                    'StdOut': 'test2\n',
                    'StdErr': ''
                }
            ],
            'Status': 'error'
            }
        },
        'Annotations': None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 1. attempt -> RC: 1',
        'Labels': {
            "command": "echo 'test2' && exit 1",
            "name": "test2"
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 2. attempt -> RC: 1',
        'Labels': {
            "command": "echo 'test2' && exit 1",
            "name": "test2"
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 3. attempt -> RC: 1',
        'Labels': {
            "command": "echo 'test2' && exit 1",
            "name": "test2"
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 4. attempt -> RC: 1',
        'Labels': {
            "command": "echo 'test2' && exit 1",
            "name": "test2"
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'result',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'result': {
                'Task': {
                    'Name': 'test3',
                    'Command': "echo 'test3' && exit 3"
                },
                'Requested': -666,
                'Requestor': '%s-scheduler' % socket.gethostname(),
                'Executor': '%s-executor' % socket.gethostname(),
                'Attempts': [
                    {
                        'Executed': -666,
                        'Duration': -666,
                        'ReturnCode': 3,
                        'StdOut': 'test3\n',
                        'StdErr': ''
                    }
                ],
                "Status": "warning"
            }
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 1. attempt -> RC: 3',
        'Labels': {
            "command": "echo 'test3' && exit 3",
            "name": "test3"
        },
        "Annotations": None
    },
    {
        'Index': '',
        'Type': 'result',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': '',
        'Labels': {
            'result': {
                'Task': {
                    'Name': 'test4',
                    'Command': "echo 'test3'"
                },
                'Requested': -666,
                'Requestor': '%s-scheduler' % socket.gethostname(),
                'Executor': '%s-executor' % socket.gethostname(),
                'Attempts': [
                    {
                        'Executed': -666,
                        'Duration': -666,
                        'ReturnCode': 0,
                        'StdOut': 'test4\n',
                        'StdErr': ''
                    }
                ],
                "Status": "success"
            }
        },
        "Annotations": None
    },
    {
        'Index': 'agentlogs-%s.%s' % (socket.gethostname().replace("-", "_"),
                                      datetime.datetime.now().strftime("%Y.%m.%d")),
        'Type': 'log',
        'Publisher': '%s-executor' % socket.gethostname(),
        'Severity': 2,
        'Message': 'Task execution request fulfilled. 1. attempt -> RC: 0',
        'Labels': {
            "command": "echo 'test4'",
            "name": "test4"
        },
        "Annotations": None
    },
]

def clean_values(input_dict, keys, value):
    for key, val in input_dict.items():
        if isinstance(val, dict):
            clean_values(val, keys, value)
        if isinstance(val, list):
            for i in val:
                if isinstance(val, dict):
                    clean_values(val, keys, value)
        if key in keys:
            input_dict[key] = value

def validate_event(evt_str):
    evt = json.loads(evt_str)
    # clean values which are impossible to mock before runtime
    clean_values(evt, ['Executed', 'Duration', 'Requested'], -666)
    clean_values(evt, ['Executed', 'Duration', 'Requested'], -666)
    if evt in expected:
        expected.remove(evt)
        return True
    return False

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
