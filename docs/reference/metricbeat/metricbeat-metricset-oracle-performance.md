---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-metricset-oracle-performance.html
---

% This file is generated! See scripts/docs_collector.py

# Oracle performance metricset [metricbeat-metricset-oracle-performance]

`performance` Metricset includes performance related events that might be correlated between them. It contains mainly cursor and cache based data and can generate 3 types of events.


## Required database access [_required_database_access]

To ensure that the module has access to the appropriate metrics, the module requires that you configure a user with access to the following tables:

* V$BUFFER_POOL_STATISTICS
* v$sesstat
* v$statname
* v$session
* v$sysstat
* V$LIBRARYCACHE


## Description of fields [_description_of_fields]

* **machine**: Operating system machine name
* **buffer_pool**: Name of the buffer pool in the instance
* **username**: Oracle username
* **io_reloads**: Reloads / Pins ratio. A Reload is any PIN of an object that is not the first PIN performed since the object handle was created, and which requires loading the object from disk. Pins are the number of times a PIN was requested for objects of this namespace.
* **lock_requests**: Average of the ratio between *gethits* and *gets* being *Gethits* the number of times an object’s handle was found in memory and *gets* the number of times a lock was requested for objects of this namespace.
* **pin_requests**: Average of all pinhits/pins ratios being *PinHits* the number of times all of the metadata pieces of the library object were found in memory and *pins* the number of times a PIN was requested for objects of this namespace.
* **cache.buffer.hit.pct**: The cache hit ratio of the specified buffer pool.
* **cache.physical_reads**: Physical reads
* **cache.get.consistent**: Consistent gets statistic
* **cache.get.db_blocks**: Database blocks gotten
* **cursors.avg**: Average cursors opened by username and machine
* **cursors.max**: Max cursors opened by username and machine
* **cursors.total**: Total opened cursors by username and machine
* **cursors.opened.current**: Total number of current open cursors
* **cursors.opened.total**: Total number of cursors opened since the instance started
* **cursors.parse.real**: Real number of parses that occurred: session cursor cache hits - parse count (total)
* **cursors.parse.total**: Total number of parse calls (hard and soft). A soft parse is a check on an object already in the shared pool, to verify that the permissions on the underlying object have not changed.
* **cursors.session.cache_hits**: Number of hits in the session cursor cache. A hit means that the SQL statement did not have to be reparsed.
* **cursors.cache_hit.pct**: Ratio of session cursor cache hits from total number of cursors


## Example events [_example_events]

Instance based cursors data:

```
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "oracle.performance",
        "duration": 115000,
        "module": "oracle"
    },
    "metricset": {
        "name": "performance"
    },
    "oracle": {
        "performance": {
            "cursors": {
                "opened": {
                    "current": 7,
                    "total": 6225
                },
                "parse": {
                    "real": 1336,
                    "total": 3684
                },
                "session": {
                    "cache_hits": 5020
                },
                "total": {
                    "cache_hit": {
                        "pct": 0.8064257028112449
                    }
                }
            },
            "io_reloads": 0.0013963503027202182,
            "lock_requests": 0.5725039956419224,
            "pin_requests": 0.7780581056654354
        }
    },
    "service": {
        "address": "oracle://sys:passwordlocalhost/ORCLPDB1.localdomain",
        "type": "oracle"
    }
}
```

Cursor data aggregated by username and machine:

```
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "oracle.performance",
        "duration": 115000,
        "module": "oracle"
    },
    "metricset": {
        "name": "performance"
    },
    "oracle": {
        "performance": {
            "cursors": {
                "avg": 0.625,
                "max": 17,
                "total": 25
            },
            "machine": "2ed9ac3a4c3d",
            "username": "Unknown"
        }
    },
    "service": {
        "address": "oracle://sys:passwordlocalhost/ORCLPDB1.localdomain",
        "type": "oracle"
    }
}
```

Cache data:

```
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "oracle.performance",
        "duration": 115000,
        "module": "oracle"
    },
    "metricset": {
        "name": "performance"
    },
    "oracle": {
        "performance": {
            "buffer_pool": "DEFAULT",
            "cache": {
                "buffer": {
                    "hit": {
                        "pct": 0.9510712759136568
                    }
                },
                "get": {
                    "consistent": 358125,
                    "db_blocks": 16195
                },
                "physical_reads": 18315
            }
        }
    },
    "service": {
        "address": "oracle://sys:passwordlocalhost/ORCLPDB1.localdomain",
        "type": "oracle"
    }
}
```

## Fields [_fields]

For a description of each field in the metricset, see the [exported fields](/reference/metricbeat/exported-fields-oracle.md) section.

Here is an example document generated by this metricset:

```json
{
    "@timestamp": "2017-10-12T08:05:34.853Z",
    "event": {
        "dataset": "oracle.performance",
        "duration": 115000,
        "module": "oracle"
    },
    "metricset": {
        "name": "performance",
        "period": 10000
    },
    "oracle": {
        "performance": {
            "cursors": {
                "cache_hit": {
                    "pct": 0.8215208034433286
                },
                "opened": {
                    "current": 32,
                    "total": 125460
                },
                "parse": {
                    "real": 39150,
                    "total": 63918
                },
                "session": {
                    "cache_hits": 103068
                }
            },
            "io_reloads": 0.009607787973500542,
            "lock_requests": 0.5939075233457263,
            "pin_requests": 0.7450330613301921
        }
    },
    "service": {
        "address": "localhost:32769",
        "type": "oracle"
    }
}
```
