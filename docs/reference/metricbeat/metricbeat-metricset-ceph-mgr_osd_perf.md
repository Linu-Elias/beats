---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-ceph-mgr_osd_perf.html
---

% This file is generated! See scripts/docs_collector.py

# Ceph mgr_osd_perf metricset [metricbeat-metricset-ceph-mgr_osd_perf]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This is the `mgr_osd_perf` metricset of the Ceph module.

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-ceph.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "ceph": {
        "mgr_osd_perf": {
            "id": 0,
            "stats": {
                "commit_latency_ms": 23,
                "apply_latency_ms": 23,
                "commit_latency_ns": 23000000,
                "apply_latency_ns": 23000000
            }
        }
    },
    "event": {
        "dataset": "ceph.mgr_osd_perf",
        "duration": 115000,
        "module": "ceph"
    },
    "metricset": {
        "name": "mgr_osd_perf"
    },
    "service": {
        "address": "127.0.0.1:8003",
        "type": "ceph"
    }
}
```
