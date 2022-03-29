# Metrics exporter

PVC metrics exporter designed to run alongside an Ondat instance.

It knows how to extract the relevant information about Ondat volumes and
makes it available as a Prometheus scraping endpoint.

Disks not owned by Ondat are ignored.

<p align="center">
    <img src="https://user-images.githubusercontent.com/26963810/160568112-1f1f6929-9fd8-428f-b03b-c7c6fdf965d2.png"
         alt="preview-overview-architecture" />
</p>

## References

- [Prometheus docs](https://prometheus.io/docs/introduction/overview/)
- [Prometheus guidelines on writting exporters](https://prometheus.io/docs/instrumenting/writing_exporters/)
- [node_exporter github](https://github.com/prometheus/node_exporter)
- [Format of /proc/diskstats](https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats)
- [About /proc/1/mounts](https://man7.org/linux/man-pages/man5/fstab.5.html)
- [More on statfs syscall](https://man7.org/linux/man-pages/man2/statfs.2.html)
