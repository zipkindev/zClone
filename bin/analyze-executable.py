#!/usr/bin/env python3
"""Measure code in stripped Windows Go executables using their runtime function tables."""
import argparse
from collections import defaultdict
import json
from pathlib import Path
import struct


def analyze(path):
    data = path.read_bytes()
    if data[:2] != b'MZ':
        raise ValueError('Expected a Windows PE executable')
    pe = struct.unpack_from('<I', data, 60)[0]
    if data[pe:pe + 4] != b'PE\0\0' or struct.unpack_from('<H', data, pe + 24)[0] != 0x20b:
        raise ValueError('Expected a 64-bit PE executable')
    count = struct.unpack_from('<H', data, pe + 6)[0]
    table = pe + 24 + struct.unpack_from('<H', data, pe + 20)[0]
    sections = {}
    for index in range(count):
        at = table + index * 40
        name = data[at:at + 8].split(b'\0')[0].decode()
        virtual, address, size, offset = struct.unpack_from('<IIII', data, at + 8)
        sections[name] = {'bytes': size, 'virtual_bytes': virtual,
                          'data': data[offset:offset + size]}
    readonly = sections['.rdata']['data']
    # Go 1.20+ uses this header; all source builds in this profile use Go 1.27.
    headers = [readonly.find(bytes.fromhex('f1ffffff0000') + bytes([quantum, 8]))
               for quantum in [1, 2, 4]]
    matches = [offset for offset in headers if offset >= 0]
    if not matches:
        raise ValueError('Unsupported Go runtime header')
    base = min(matches)
    nfunc, nfiles, text, names, units, files, pc, pcln = struct.unpack_from('<8Q', readonly, base + 8)
    if not 0 < nfunc < 1000000 or not names < units <= files <= pc <= pcln < len(readonly):
        raise ValueError('Unsupported or invalid Go runtime function table')
    ftable = base + pcln
    functions = []
    for index in range(nfunc):
        entry, info = struct.unpack_from('<II', readonly, ftable + index * 8)
        end = struct.unpack_from('<I', readonly, ftable + (index + 1) * 8)[0]
        if not 0 <= entry <= end <= sections['.text']['bytes']:
            raise ValueError('Function falls outside the executable code section')
        name_offset = struct.unpack_from('<i', readonly, ftable + info + 4)[0]
        start = base + names + name_offset
        name = readonly[start:readonly.index(b'\0', start)].decode()
        functions.append({'name': name, 'bytes': end - entry})
    packages = defaultdict(int)
    for function in functions:
        head = function['name'].split('[', 1)[0]
        directory, _, tail = head.rpartition('/')
        package = (directory + '/' if directory else '') + tail.split('.', 1)[0]
        packages[package] += function['bytes']
    s3 = 'github.com/aws/aws-sdk-go-v2/service/s3.(*Client).'
    return {'path': str(path), 'file_bytes': len(data),
            'sections': {name: {key: value for key, value in section.items() if key != 'data'}
                         for name, section in sections.items()},
            'function_count': nfunc, 'function_code_bytes': sum(f['bytes'] for f in functions),
            'function_names_bytes': units - names, 'pc_mapping_bytes': pcln - pc,
            'generic_function_count': sum('[' in f['name'] and not f['name'].startswith('type:') for f in functions),
            'generic_function_bytes': sum(f['bytes'] for f in functions if '[' in f['name'] and not f['name'].startswith('type:')),
            's3_client_methods': [f['name'][len(s3):] for f in functions
                                  if f['name'].startswith(s3) and '.' not in f['name'][len(s3):]],
            'go_json_code_bytes': sum(f['bytes'] for f in functions
                                      if f['name'].startswith('github.com/goccy/go-json/')),
            'packages': dict(sorted(packages.items(), key=lambda item: -item[1])),
            'largest_functions': sorted(functions, key=lambda item: -item['bytes'])[:30]}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--link-dependencies', type=Path)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    report = {'baseline': analyze(args.baseline), 'candidate': analyze(args.candidate),
              'notes': ['Function sizes include alignment to the next function.',
                        'Library code sizes exclude associated data and metadata.',
                        'Generic instantiations are not inherently unnecessary code.',
                        'Linker trace edges record first-reach parents, not a complete runtime call graph.']}
    report['saved_bytes'] = {key: report['baseline'][key] - report['candidate'][key]
                             for key in ['file_bytes', 'function_code_bytes', 'function_count',
                                         'function_names_bytes', 'pc_mapping_bytes', 'go_json_code_bytes']}
    if args.link_dependencies:
        needles = ['text/template.(*state).evalFieldChain ->',
                   'text/template.(*state).evalField <ReflectMethod> -> reflect.Value.MethodByName',
                   'type:zclone/backend/s3.Fs <UsedInIface> -> type:*github.com/aws/aws-sdk-go-v2/service/s3.Client',
                   '<UsedInIface> -> github.com/aws/aws-sdk-go-v2/service/s3.(*Client).PutBucketAnalyticsConfiguration',
                   'github.com/goccy/go-json.encodeRunCode -> github.com/goccy/go-json/internal/encoder/vm_color.Run']
        with args.link_dependencies.open() as stream:
            report['linker_retention_evidence'] = [line.strip() for line in stream
                                                  if any(needle in line for needle in needles)]
    args.output.parent.mkdir(parents=True, exist_ok=True)
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report['saved_bytes'], indent=2))


if __name__ == '__main__':
    main()
