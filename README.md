# Metrics exporter

PVC metrics exporter designed to run alongside an Ondat instance.

It knows where and how to extract the relevant information about the Ondat volumes and 
makes it available as a Prometheus endpoint. Disks not owned by Ondat are ignored.

<p align="center">
    <img src="https://user-images.githubusercontent.com/26963810/158974781-f06883cc-0bdc-4c90-b24b-22da61b1cab7.png"
         alt="preview-overview-architecture" />
</p>

## References

- [Prometheus docs](https://prometheus.io/docs/introduction/overview/)
- [Prometheus guidelines on writting exporters](https://prometheus.io/docs/instrumenting/writing_exporters/)
- [node_exporter github](https://github.com/prometheus/node_exporter)
- [Format of /proc/diskstats](https://www.kernel.org/doc/Documentation/ABI/testing/procfs-diskstats)
- [About /proc/1/mounts](https://man7.org/linux/man-pages/man5/fstab.5.html)
- [More on statfs syscall](https://man7.org/linux/man-pages/man2/statfs.2.html)
