---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-system-entropy.html
---

% This file is generated! See scripts/docs_collector.py

# System entropy metricset [metricbeat-metricset-system-entropy]

This is the entropy metricset of the module system. It collects the amount of available entropy in bits. On kernel versions greater than 2.6, entropy will be out of a total pool size of 4096.

This Metricset is available on:

* linux

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-system.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "system.entropy",
        "duration": 115000,
        "module": "system"
    },
    "metricset": {
        "name": "entropy"
    },
    "service": {
        "type": "system"
    },
    "system": {
        "entropy": {
            "available_bits": 2801,
            "pct": 0.683837890625
        }
    }
}
```
