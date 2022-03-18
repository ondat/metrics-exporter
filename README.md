# Metrics exporter

PVC metrics exporter designed to run alongside an Ondat instance.

It knows where and how to extract the relevant information about the Ondat volumes and makes it available as a Prometheus endpoint. Disks not owned by Ondat are ignored.

<p align="center">
<img src="https://user-images.githubusercontent.com/26963810/157466080-90678c58-5657-4341-a6fa-eb5e9850af58.png" alt="preview-overview-architecture" />
</p>


All the disk metrics are processed following the node_exporter's implementation.

We also include metrics about the scrape itself:
- `ondat_scrape_collector_success`
- `ondat_scrape_collector_duration_seconds`

## References
 - [Prometheus docs](https://prometheus.io/docs/introduction/overview/)
 - [Prometheus guidelines on writting exporters](https://prometheus.io/docs/instrumenting/writing_exporters/)
 - [node_exporter github](https://github.com/prometheus/node_exporter)
 - [Format of /proc/diskstats](https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats)
