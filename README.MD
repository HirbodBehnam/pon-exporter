# PON Exporter

Export PON status of your router as Prometheus metrics. I wrote this application for SHG-410AC model.

## Usage

The most important argument is the admin password which is the admin password of the router dashboard. For me, it defaults to `admin410`. You can also set the listening port and update interval time. Here is the help of program:

```
Usage of pon-exporter:
  -admin-password string
        The admin password of dashboard. (default "admin410")
  -listen-address string
        The address to listen on for HTTP requests. (default ":8080")
  -update-interval int
        Update interval in seconds. (default 15)
```