package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	cni "github.com/containerd/go-cni"
	"github.com/phosae/mockctrd/annotations"
	"k8s.io/apimachinery/pkg/api/resource"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type Runtime struct {
	// NetworkPluginConfDir is a directory containing the CNI network information for the runtime class.
	NetworkPluginConfDir string `toml:"cni_conf_dir" json:"cniConfDir"`
	// NetworkPluginMaxConfNum is the max number of plugin config files that will
	// be loaded from the cni config directory by go-cni. Set the value to 0 to
	// load all config files (no arbitrary limit). The legacy default value is 1.
	NetworkPluginMaxConfNum int `toml:"cni_max_conf_num" json:"cniMaxConfNum"`
}

// CniConfig contains toml config related to cni
type CniConfig struct {
	// NetworkPluginBinDir is the directory in which the binaries for the plugin is kept.
	NetworkPluginBinDir string `toml:"bin_dir" json:"binDir"`
	// NetworkPluginConfDir is the directory in which the admin places a CNI conf.
	NetworkPluginConfDir string `toml:"conf_dir" json:"confDir"`
	// NetworkPluginMaxConfNum is the max number of plugin config files that will
	// be loaded from the cni config directory by go-cni. Set the value to 0 to
	// load all config files (no arbitrary limit). The legacy default value is 1.
	NetworkPluginMaxConfNum int `toml:"max_conf_num" json:"maxConfNum"`
	// IPPreference specifies the strategy to use when selecting the main IP address for a pod.
	//
	// Options include:
	// * ipv4, "" - (default) select the first ipv4 address
	// * ipv6 - select the first ipv6 address
	// * cni - use the order returned by the CNI plugins, returning the first IP address from the results
	IPPreference string `toml:"ip_pref" json:"ipPref"`
}

const (
	defaultNetworkPlugin = "default"
	// networkAttachCount is the minimum number of networks the PodSandbox
	// attaches to
	networkAttachCount = 2
)

/*
cat /etc/cni/net.d/10-macvlan.conflist

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
*/
func main() {
	var (
		i         cni.CNI
		err       error
		netnspath string
		dryrun    bool = true

		conf = CniConfig{
			NetworkPluginBinDir:     "/opt/cni/bin",
			NetworkPluginConfDir:    "/etc/cni/net.d",
			NetworkPluginMaxConfNum: 1,
		}
	)

	if cnipath := os.Getenv("CNI_BIN"); cnipath != "" {
		conf.NetworkPluginBinDir = cnipath
	}
	if cniconf := os.Getenv("CNI_CONF"); cniconf != "" {
		conf.NetworkPluginConfDir = cniconf
	}
	if netnspath = os.Getenv("CNI_NETNS"); netnspath == "" {
		netnspath = "/var/run/netns/zenx"
	}
	if os.Getenv("DRYRUN") == "false" {
		dryrun = false
	}

	// Pod needs to attach to at least loopback network and a non host network,
	// hence networkAttachCount is 2. If there are more network configs the
	// pod will be attached to all the networks but we will only use the ip
	// of the default network interface as the pod IP.
	i, err = cni.New(cni.WithMinNetworkCount(networkAttachCount),
		cni.WithPluginConfDir(conf.NetworkPluginConfDir),
		cni.WithPluginMaxConfNum(conf.NetworkPluginMaxConfNum),
		cni.WithPluginDir([]string{conf.NetworkPluginBinDir}))
	if err != nil {
		panic(fmt.Errorf("failed to initialize cni: %w", err))
	}

	cniopts, err := cniNamespaceOpts("sandbox_id", &runtime.PodSandboxConfig{
		Metadata: &runtime.PodSandboxMetadata{
			Name:      "pod",
			Uid:       "xid-123",
			Namespace: "default",
		},
		Hostname:     "mynode",
		LogDirectory: "",
		DnsConfig:    &runtime.DNSConfig{},
		PortMappings: []*runtime.PortMapping{
			{
				Protocol:      runtime.Protocol_TCP,
				ContainerPort: 80,
				HostPort:      18080,
				//HostIp:               "",
			},
		},
		Labels: map[string]string{},
		Annotations: map[string]string{
			"kubernetes.io/ingress-bandwidth": "200Mi",
			"kubernetes.io/egress-bandwidth":  "100Mi",
		},
		Linux: &runtime.LinuxPodSandboxConfig{},
	})
	if err != nil {
		panic(err)
	}

	// CNI_ARGS
	// cniopts = append(cniopts, cni.WithArgs("IP", "192.168.64.211"))

	err = i.Load([]cni.Opt{cni.WithLoNetwork, cni.WithDefaultConf}...)
	if err != nil {
		panic(err)
	}

	//ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	time.Sleep(time.Nanosecond)
	defer cancel()

	defer func() {
		if dryrun {
			err = i.Remove(ctx, "sanbox_id", netnspath, cniopts...)
			if err != nil {
				panic(err)
			}
		}
	}()

	ret, err := i.Setup(ctx, "sanbox_id", netnspath, cniopts...)
	if err != nil {
		panic(err)
	}
	b, _ := json.MarshalIndent(ret, "", "  ")
	fmt.Println(string(b))
}

// cniNamespaceOpts get CNI namespace options from sandbox config.
func cniNamespaceOpts(id string, config *runtime.PodSandboxConfig) ([]cni.NamespaceOpts, error) {
	opts := []cni.NamespaceOpts{
		cni.WithLabels(toCNILabels(id, config)),
		cni.WithCapability(annotations.PodAnnotations, config.Annotations),
	}

	portMappings := toCNIPortMappings(config.GetPortMappings())
	if len(portMappings) > 0 {
		opts = append(opts, cni.WithCapabilityPortMap(portMappings))
	}

	// Will return an error if the bandwidth limitation has the wrong unit
	// or an unreasonable value see validateBandwidthIsReasonable()
	bandWidth, err := toCNIBandWidth(config.Annotations)
	if err != nil {
		return nil, err
	}
	if bandWidth != nil {
		opts = append(opts, cni.WithCapabilityBandWidth(*bandWidth))
	}

	dns := toCNIDNS(config.GetDnsConfig())
	if dns != nil {
		opts = append(opts, cni.WithCapabilityDNS(*dns))
	}

	return opts, nil
}

// toCNILabels adds pod metadata into CNI labels.
func toCNILabels(id string, config *runtime.PodSandboxConfig) map[string]string {
	return map[string]string{
		"K8S_POD_NAMESPACE":          config.GetMetadata().GetNamespace(),
		"K8S_POD_NAME":               config.GetMetadata().GetName(),
		"K8S_POD_INFRA_CONTAINER_ID": id,
		"K8S_POD_UID":                config.GetMetadata().GetUid(),
		"IgnoreUnknown":              "1",
	}
}

// toCNIBandWidth converts CRI annotations to CNI bandwidth.
func toCNIBandWidth(annotations map[string]string) (*cni.BandWidth, error) {
	ingress, egress, err := ExtractPodBandwidthResources(annotations)
	if err != nil {
		return nil, fmt.Errorf("reading pod bandwidth annotations: %w", err)
	}

	if ingress == nil && egress == nil {
		return nil, nil
	}

	bandWidth := &cni.BandWidth{}

	if ingress != nil {
		bandWidth.IngressRate = uint64(ingress.Value())
		bandWidth.IngressBurst = math.MaxUint32
	}

	if egress != nil {
		bandWidth.EgressRate = uint64(egress.Value())
		bandWidth.EgressBurst = math.MaxUint32
	}

	return bandWidth, nil
}

// toCNIPortMappings converts CRI port mappings to CNI.
func toCNIPortMappings(criPortMappings []*runtime.PortMapping) []cni.PortMapping {
	var portMappings []cni.PortMapping
	for _, mapping := range criPortMappings {
		if mapping.HostPort <= 0 {
			continue
		}
		portMappings = append(portMappings, cni.PortMapping{
			HostPort:      mapping.HostPort,
			ContainerPort: mapping.ContainerPort,
			Protocol:      strings.ToLower(mapping.Protocol.String()),
			HostIP:        mapping.HostIp,
		})
	}
	return portMappings
}

// toCNIDNS converts CRI DNSConfig to CNI.
func toCNIDNS(dns *runtime.DNSConfig) *cni.DNS {
	if dns == nil {
		return nil
	}
	return &cni.DNS{
		Servers:  dns.GetServers(),
		Searches: dns.GetSearches(),
		Options:  dns.GetOptions(),
	}
}

// ExtractPodBandwidthResources extracts the ingress and egress from the given pod annotations
func ExtractPodBandwidthResources(podAnnotations map[string]string) (ingress, egress *resource.Quantity, err error) {
	if podAnnotations == nil {
		return nil, nil, nil
	}
	str, found := podAnnotations["kubernetes.io/ingress-bandwidth"]
	if found {
		ingressValue, err := resource.ParseQuantity(str)
		if err != nil {
			return nil, nil, err
		}
		ingress = &ingressValue
		if err := validateBandwidthIsReasonable(ingress); err != nil {
			return nil, nil, err
		}
	}
	str, found = podAnnotations["kubernetes.io/egress-bandwidth"]
	if found {
		egressValue, err := resource.ParseQuantity(str)
		if err != nil {
			return nil, nil, err
		}
		egress = &egressValue
		if err := validateBandwidthIsReasonable(egress); err != nil {
			return nil, nil, err
		}
	}
	return ingress, egress, nil
}

var minRsrc = resource.MustParse("1k")
var maxRsrc = resource.MustParse("1P")

func validateBandwidthIsReasonable(rsrc *resource.Quantity) error {
	if rsrc.Value() < minRsrc.Value() {
		return fmt.Errorf("resource is unreasonably small (< 1kbit)")
	}
	if rsrc.Value() > maxRsrc.Value() {
		return fmt.Errorf("resource is unreasonably large (> 1Pbit)")
	}
	return nil
}

func cniLoadOptions() []cni.Opt {
	return []cni.Opt{cni.WithLoNetwork, cni.WithDefaultConf}
}
