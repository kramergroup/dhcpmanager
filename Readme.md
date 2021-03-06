# DHCPManager

A restful HTTP service providing management of DHCP-based IPs. DHCPManager can:

-   Obtain IP addresses from a DHCP server
-   Renew leases until the IP is returned
-   Provide configuration and status information

The service is meant to operate in conjunction with [metallb](https://metallb.universe.tf/), a load balancer
for bare-metal Kubernetes cluster, but currently _requires a custom fork_ fund on [github.com](https://github.com/kramergroup/metallb/tree/feature-dhcp).

## API

The service provides a restful API for managing IPs:

| URL          | Method | Body                            | Description                                    |
| ------------ | ------ | ------------------------------- | ---------------------------------------------- |
| `/v1/config` | GET    |                                 | Obtain service configuration                   |
| `/v1/status` | GET    |                                 | Obtain service status (not used in metallb)    |
| `/v1/ip`     | POST   | `{"service":"namespace/svc"}`   | Request a new IP for `service`                 |
|              | DELETE | `{"ip":"xxx.xxx.xxx.xxx"}`      | Return an IP                                   |
| `/v1/mac`    | POST   | `{"macs":["xx.xx.xx.xx.xx.xx"]` | Provide a list of hardware addresses to use    |
|              | DELETE | `{"macs":["xx.xx.xx.xx.xx.xx"]` | Remove a list of hardware addresses from usage |

### Obtaining an addresses

A new IP is obtained by using a `POST` against the `/v1/ip` endpoint:

    curl -X POST -d '{"service":"name"}' http://<server>/ip

The `service` parameter will be used as hostname for the DHCP request. If the DHCP server is tied to a DNS server,
`name.<domain>` will resolve to the provided IP.

A typical response returns the IP, status, and a reference ID:

```json
{"ip":"192.168.1.100","id":"d24b92f1-2e40-4c2d-b074-1c438ae31e78","status":"success"}
```

### Returning addresses

Addresses should be returned to the service after use to save resources.

An address is returned with a `DELETE` request against `/v1/ip`

    curl -X DELETE -d '{"ip":"192.168.1.100"}' http://<server>/ip

A successful response returns `status=success` amongst other information:

```json
{"ip":"192.168.1.100","id":"d24b92f1-2e40-4c2d-b074-1c438ae31e78","status":"success"}
```

### Managing MAC addresses

MAC addresses can be provided with the configuration file (see below) or dynamically
added/removed using the `/v1/mac` endpoint.

Adding MACs:

```bash
curl -X POST -d '{"macs": ["xx.xx.xx.xx.xx.xx"]}' http://<server>/ip
```

Removing MACs:

```bash
curl -X DELETE -d '{"macs": ["xx.xx.xx.xx.xx.xx"]}' http://<server>/ip
```

> MACs will only be removed if not in use. The response will contain a list of
> omitted addresses.

### Monitoring

The service provides two endpoints to monitor configuration and state:

| endpoint     | method | purpose                                            |
| ------------ | ------ | -------------------------------------------------- |
| `/v1/config` | `GET`  | returns configuration information                  |
| `/v1/status` | `GET`  | provides status information such as current leases |

## Configuration

The service is configured via `/etc/dhcpmanager/dhcpmanager.toml` and environment variables.

| Variable          | Environment Variable   | Default         | Comment                                                    |
| ----------------- | ---------------------- | --------------- | ---------------------------------------------------------- |
| etcd              | DHCP_ETCD              | `["etcd:2379"]` | Array of etcd endpoints                                    |
| interface         | DHCP_INTERFACE         | `eth0`          | The network interface used for DHCP requests               |
| manage-interfaces | DHCP_MANAGE_INTERFACES | `true`          | Manage creation of network interfaces                      |
| assign-interfaces | DHCP_ASSIGN_INTERFACES | `false`         | Assign IPs to interfaces                                   |
| macs              | DHCP_MACS              | `[]`            | Array of MAC addresses used for virtual network interfaces |

A typical configuration file looks like:

```toml
etcd = [ "etcd:2379" ]
interface = "eth0"
manage-interfaces = true
assign-interfaces = false
macs = [
  "56:6A:E2:0B:01:8D",
  "30:BA:33:C2:E3:C2",
]
```

### Avoiding secondary IPs

If your system is setup to automatically obtain IPs for network interfaces, you
might obtain a second IP address, because the system-wide DHCP client requests
one as soon as the interface is created. This can be avoided by configuring
the system to not use DHCP for interfaces that are named `vf-*`. How this is achieved
depends on your system.

#### Systemd

A `systemd`-based OS can be configured by adding a `network` file for the `vf-*` interfaces.

Create a file `/etc/systemd/network/30-vf.network` (or similarly named) with content

```
[Match]
Name=vf-*

[Network]
DHCP=no
```

and restart the network daemon:

```bash
systemctl restart systemd-networkd
```

## Status

This code is functional, but not widely tested. We mainly use it for our own small-scale
research bare-metals Kubernetes cluster. We consider it alpha software. There are several
features missing, most notably:

-   Security is all but missing. We have plans for HTTPS with client authentication as well as OAUTH2, but this is low priority for us.
-   Webhooks for life-cycle events are also missing. These would be good for better integration with custom environments.

### Issues

-   We currently 'leak' MAC addresses if they are not properly returned (in our setup, we rely on metallb to return IP addresses).

## Deployment

The service consists of two components:

-   _Controller_ manages network interfaces and DHCP communication
-   _Apiserver_ provides the HTTP endpoint API into the service

There are two requirements:

-   an [etcd3](https://github.com/coreos/etcd) key-value store to persist state
-   _Controller_ has to run on the host network to setup network interfaces

Sample deployment configurations are provided for Kubernetes and docker-compose.
