The System `process` metricset provides process statistics. One document is provided for each process.

This metricset is available on:

* FreeBSD
* Linux
* macOS
* Windows


## Configuration [_configuration_12]

**`processes`**
:   When the `process` metricset is enabled, you can use the `processes` option to define a list of regexp expressions to filter the processes that are reported. For more complex filtering, you should use the `processors` configuration option. See [Processors](/reference/metricbeat/filtering-enhancing-data.md) for more information.

    The following example config returns metrics for all processes:

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      processes: ['.*']
    ```


**`process.cgroups.enabled`**
:   When the `process` metricset is enabled, you can use this boolean configuration option to disable cgroup metrics. By default cgroup metrics collection is enabled.

    The following example config disables cgroup metrics on Linux.

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.cgroups.enabled: false
    ```


**`process.cmdline.cache.enabled`**
:   This metricset caches the command line args for a running process by default. This means if you alter the command line for a process while this metricset is running, these changes are not detected. Caching can be disabled by setting `process.cmdline.cache.enabled: false` in the configuration.

**`process.env.whitelist`**
:   This metricset can collect the environment variables that were used to start the process. This feature is available on Linux, Darwin, and FreeBSD. No environment variables are collected by default because they could contain sensitive information. You must configure the environment variables that you wish to collect by specifying a list of regular expressions that match the variable name.

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.env.whitelist:
      - '^PATH$'
      - '^SSH_.*'
    ```


**`process.include_cpu_ticks`**
:   By default the cumulative CPU tick values are not reported by this metricset (only percentages are reported). Setting this option to true will enable the reporting of the raw CPU tick values (for user, system, and total CPU time).

    ```yaml
    metricbeat.modules:
    - module: system
      metricsets: ["process"]
      process.include_cpu_ticks: true
    ```


**`process.include_per_cpu`**
:   By default metrics per cpu are reported when available. Setting this option to false will disable the reporting of these metrics.

**`process.include_top_n`**
:   These options allow you to filter out all processes that are not in the top N by CPU or memory, in order to reduce the number of documents created. If both the `by_cpu` and `by_memory` options are used, the union of the two sets is included.

**`process.include_top_n.enabled`**
:   Set to false to disable the top N feature and include all processes, regardless of the other options. The default is `true`, but nothing is filtered unless one of the other options (`by_cpu` or `by_memory`) is set to a non-zero value.

**`process.include_top_n.by_cpu`**
:   How many processes to include from the top by CPU. The processes are sorted by the `system.process.cpu.total.pct` field. The default is 0.

**`process.include_top_n.by_memory`**
:   How many processes to include from the top by memory. The processes are sorted by the `system.process.memory.rss.bytes` field. The default is 0.


## Monitoring Hybrid Hierarchy Cgroups [_monitoring_hybrid_hierarchy_cgroups]

The process metricset supports both V1 and V2 (sometimes called unfied) cgroups controllers. However, on systems that are running a hybrid hierarchy, with both V1 and V2 controllers, metricbeat will only report one of the hierarchies for a given process. Is a process has both V1 and V2 hierarchies associated with it, metricbeat will check to see if the process is attached to any V2 controllers. If it is, it will report cgroups V2 metrics. If not, it will report V1 metrics.

A workaround is also required if metricbeat is running inside docker on a hybrid system. Within docker, metricbeat won’t be able to see any V2 cgroups components. If you wish to monitor cgroups V2 from within docker on a hybrid system, you must mount the unified sysfs hierarchy (usually `/sys/fs/cgroups/unified`) inside the container, and then use `system.hostfs` to specify the filesystem root within the container.
