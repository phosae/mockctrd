## about this repo

mock interaction between containerd and K8s CNI plugins...

setup netns, prepare CNI plugins and CNI plugin configs
```
{
ip netns add zenx

cat << EOF | tee /etc/cni/net.d/10-macvlan.conflist
{
  "cniVersion": "0.3.1",
  "name": "debugcni",
  "plugins": [
  {
    "type": "macvlan",
    "name": "macvlan",
    "master": "enp0s1",
    "mode": "bridge",
    "ipam":{
        "type": "host-local",
        "ranges": [
          [{"subnet": "192.168.64.0/24"}]
        ],
        "gateway": "192.168.64.1",
        "routes": [{"dst": "0.0.0.0/0"}],
        "dataDir": "/tmp/host-local"
    }
  },
  {"type": "portmap", "snat": true, "capabilities": {"portMappings": true}}
  ]
}
EOF

docker run --rm -v /opt/cni/bin:/out -e CNI_BIN_DST=/out zengxu/cni-copier:221215-ec76e3c
}
``

```
DRYRUN=true CNI_NETNS=/var/run/zenx ./mockctrd
```
- if DRYRUN set to false, cmdDel will not been called when exit
- u can use CNI_NETNS env to set netns