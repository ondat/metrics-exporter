# Metrics exporter

PVC metrics exporter designed to run alongside an Ondat instance.

It knows where and how to extract the relevant information about the Ondat volumes and makes it available as a Prometheus endpoint. Disks not owned by Ondat are ignored.

<p align="center">
<img src="https://user-images.githubusercontent.com/26963810/157465141-7b4bd155-4a0e-4666-ad64-b060d62fab41.png" alt="preview-overview-architecture" />
</p>

All the disk metrics are processed following the node_exporter's implementation


## References
 - [Prometheus docs](https://prometheus.io/docs/introduction/overview/)
 - [Prometheus guidelines on writting exporters](https://prometheus.io/docs/instrumenting/writing_exporters/)
 - [node_exporter github](https://github.com/prometheus/node_exporter)
 - [Format of /proc/diskstats](https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats)
