package src

import (
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/input-provider-aliyun/src/compute"
	"github.com/hahaps/input-provider-aliyun/src/network"
	"github.com/hahaps/input-provider-aliyun/src/storage"
)

var ResourceMap = map[string]input.Resource {
	"Server": &compute.Server{},
	"ServerMetric": &compute.ServerMetric{},
	"Image": &compute.Image{},
	"Disk": &storage.Disk{},
	"Network": &network.Network{},
	"Subnet": &network.Subnet{},
	"Nic": &network.Nic{},
	"SecurityGroup": &network.SecurityGroup{},
	"FloatingIp": &network.FloatingIp{},
	"FloatingIpMetric": &network.FloatingIpMetric{},
	"InstanceBill": &InstanceBill{},
}
