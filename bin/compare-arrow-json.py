#!/usr/bin/env python3
"""Compare Arrow test binaries built with the default and standard JSON libraries."""
import argparse
import json
import os
from pathlib import Path
import re
import statistics
import subprocess


def parse(output):
    results = {}
    for line in output.splitlines():
        fields = line.split()
        if not fields or not fields[0].startswith('BenchmarkArrow'):
            continue
        name = re.sub(r'-\d+$', '', fields[0])
        results[name] = {unit: float(fields[fields.index(unit) - 1])
                         for unit in ['ns/op', 'B/op', 'allocs/op']}
    if not results:
        raise RuntimeError('No Arrow benchmarks were reported')
    return results


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--baseline', type=Path, required=True)
    parser.add_argument('--candidate', type=Path, required=True)
    parser.add_argument('--count', type=int, default=7)
    parser.add_argument('--output', type=Path, required=True)
    args = parser.parse_args()
    if args.count < 3:
        parser.error('at least three samples are required')
    args.output.parent.mkdir(parents=True, exist_ok=True)
    env = dict(os.environ, GOMAXPROCS='2', ZCLONE_CONFIG=os.devnull)
    binaries = {'baseline': args.baseline.resolve(), 'candidate': args.candidate.resolve()}
    report = {'samples': args.count, 'gomaxprocs': 2, 'benchmarks': {},
              'listing_latency_limit_percent': 20, 'listing_allocation_limit_percent': 5}
    with args.output.with_suffix('.log').open('w') as log:
        def execute(binary, *options):
            result = subprocess.run([str(binary), *options], env=env, capture_output=True,
                                    text=True, timeout=180)
            log.write(str(binary) + '\n' + result.stdout + result.stderr + '\n')
            log.flush()
            if result.returncode:
                raise RuntimeError(result.stdout + result.stderr)
            return result.stdout
        for binary in binaries.values():
            execute(binary, '-test.v', '-test.timeout', '2m')
            execute(binary, '-test.run', '^$', '-test.bench', '^BenchmarkArrow',
                    '-test.benchmem', '-test.benchtime', '100ms')
        samples = {label: {} for label in binaries}
        for trial in range(args.count):
            labels = list(binaries) if trial % 2 == 0 else list(reversed(binaries))
            for label in labels:
                output = execute(binaries[label], '-test.run', '^$', '-test.bench', '^BenchmarkArrow',
                                 '-test.benchmem', '-test.benchtime', '500ms')
                for name, values in parse(output).items():
                    samples[label].setdefault(name, []).append(values)
    if samples['baseline'].keys() != samples['candidate'].keys():
        raise RuntimeError('Benchmark inventories differ')
    failures = []
    for name in samples['baseline']:
        entry = {}
        for label in binaries:
            values = samples[label][name]
            if len(values) != args.count:
                raise RuntimeError('Missing samples for ' + name)
            entry[label] = {unit: statistics.median(v[unit] for v in values)
                            for unit in ['ns/op', 'B/op', 'allocs/op']}
            entry[label]['latency_range_ns'] = [min(v['ns/op'] for v in values),
                                                max(v['ns/op'] for v in values)]
        entry['change_percent'] = {unit: 100 * (entry['candidate'][unit] / entry['baseline'][unit] - 1)
                                   if entry['baseline'][unit] else None
                                   for unit in ['ns/op', 'B/op', 'allocs/op']}
        if name.startswith('BenchmarkArrowListing/'):
            for unit, limit in [('ns/op', 20), ('B/op', 5), ('allocs/op', 5)]:
                if entry['change_percent'][unit] is not None and entry['change_percent'][unit] > limit:
                    failures.append(name + ' ' + unit + ' regression')
        report['benchmarks'][name] = entry
    report['regressions'] = failures
    args.output.write_text(json.dumps(report, indent=2) + '\n')
    print(json.dumps(report, indent=2))
    if failures:
        raise SystemExit('; '.join(failures))


if __name__ == '__main__':
    main()
