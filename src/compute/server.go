package compute

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models/compute"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
)

var ServerSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "region",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "limit",
		Required: false,
		Type: utils.Int,
		Default: utils.DefaultLimit,
	},
	utils.Scheme{
		Param: "marker",
		Required: false,
		Type: utils.String,
		Default: "1",
	},
}

type Server struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (server *Server)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs-cn-hangzhou.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	server.credential = credential
	server.client = cli
	return nil
}

func (Server)Call(params input.Params, replay *input.Replay) error {
	server := &Server{}
	var next string
	var err error
	var servers []interface{}
	params.Args, err = utils.CheckParam(params.Args, ServerSchemes)
	if err != nil {
		return err
	}
	err = server.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	region := params.Args["region"].(string)
	query := map[string]interface{} {
		"RegionId": region,
		"CloudType": common.CloudType,
		"AccountId": server.credential.AccountId,
	}
	if next == "" {
		return nil
	}
	number, err := strconv.ParseInt(next,10,32)
	if err != nil {
		return errors.New("bad page number[marker] info")
	}
	pageNum := int32(number)
	request := &ecs20140526.DescribeInstancesRequest{
		RegionId: tea.String(region),
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := server.client.DescribeInstances(request)
	if err != nil {
		return err
	}
	for _, instance := range resp.Body.Instances.Instance {
		serv := compute.NewServerModel()
		serv.Deleted = 0
	    serv.CloudType = common.CloudType
	    serv.AccountId = server.credential.AccountId
		serv.RegionId = region
		serv.ProviderId = utils.SafeString(instance.InstanceId)
		serv.Name = utils.SafeString(instance.InstanceName)
		serv.CreateTime = utils.SafeString(instance.CreationTime)
		serv.ExpireTime = utils.SafeString(instance.ExpiredTime)
		serv.Status = utils.SafeString(instance.Status)
		serv.PayMode = utils.SafeString(instance.InstanceChargeType)
		serv.AutoRenew = false
		serv.FlavorId = utils.SafeString(instance.InstanceType)
		serv.FlavorName = utils.SafeString(instance.InstanceType)
		serv.FlavorRam = int(utils.SafeInt32(instance.Memory))
		serv.FlavorVCPU = int(utils.SafeInt32(instance.Cpu))
		serv.FlavorExtra = map[string]interface{}{
			"CPUOptions": instance.CpuOptions.String(),
			"GPUAmount": int(utils.SafeInt32(instance.GPUAmount)),
			"GPUSpec": utils.SafeString(instance.GPUSpec),
		}
		serv.ImageId = utils.SafeString(instance.ImageId)
		serv.ImageName = utils.SafeString(instance.ImageId)
		serv.ImageOsType = utils.SafeString(instance.OSType)
		serv.ImageExtra = map[string]interface{}{
			"ImageOsName": utils.SafeString(instance.OSName),
			"ImageOsNameEn": utils.SafeString(instance.OSNameEn),
		}
		if instance.VpcAttributes != nil {
			serv.PrimaryNetworkId = utils.SafeString(instance.VpcAttributes.VpcId)
			serv.PrimarySubnetId = utils.SafeString(instance.VpcAttributes.VSwitchId)
			if instance.VpcAttributes.PrivateIpAddress != nil {
				serv.PrimaryNicIp = utils.JoinStringPtr(instance.VpcAttributes.PrivateIpAddress.IpAddress, ", ")
			}
		}
		serv.PrimaryNicFloatingIp = getFloatingIp(
			instance.EipAddress, instance.PublicIpAddress)
		serv.SecondaryNics = getSecondaryNics(instance.NetworkInterfaces)
		serv.SecurityGroups = getSecurityGroups(instance.SecurityGroupIds)
		serv.Tags = getTags(instance.Tags)
		serv.Extra = map[string]interface{}{
			"StoppedMode": utils.SafeString(instance.StoppedMode),
			"DeletionProtection": utils.SafeBool(instance.DeletionProtection, false),
			"InternetChargeType": utils.SafeString(instance.InternetChargeType),
			"AutoReleaseTime": utils.SafeString(instance.AutoReleaseTime),
			"InstanceTypeFamily": utils.SafeString(instance.InstanceTypeFamily),
			"ZoneId": utils.SafeString(instance.ZoneId),
			"ResourceGroup": utils.SafeString(instance.ResourceGroupId),
		}
		serv.SetIndex()
		serv.SetChecksum()
		checked, key := serv.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		servers = append(servers, serv)
	}
	if !utils.CheckQueryKeys(query, compute.ServerModel{}) {
		return errors.New("query key is not attribute of ServerModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = servers
	return nil
}

func getSecondaryNics(nics *ecs20140526.DescribeInstancesResponseBodyInstancesInstanceNetworkInterfaces)map[string]interface{} {
	secondary := map[string]interface{}{}
	if nics == nil {
		return secondary
	}
	for _, nic := range nics.NetworkInterface {
		if *nic.Type == "Primary" {
			continue
		}
		secondary[*nic.NetworkInterfaceId] = *nic.PrimaryIpAddress
	}
	return secondary
}

func getSecurityGroups(sgs *ecs20140526.DescribeInstancesResponseBodyInstancesInstanceSecurityGroupIds) map[string]string {
	secg := map[string]string{}
	if sgs == nil {
		return secg
	}
	for _, sg := range sgs.SecurityGroupId {
		secg[*sg] = *sg
	}
	return secg
}

func getTags(tags *ecs20140526.DescribeInstancesResponseBodyInstancesInstanceTags) string {
	if tags == nil {
		return ""
	}
	tgs := ""
	l := len(tags.Tag)
	for i, tg := range tags.Tag {
		tgs += fmt.Sprintf("%v=%v", tg.TagKey, tg.TagValue)
		if i + 1 < l {
			tgs += ";"
		}
	}
	return tgs
}

func getFloatingIp(eIp *ecs20140526.DescribeInstancesResponseBodyInstancesInstanceEipAddress,
	floatingIp *ecs20140526.DescribeInstancesResponseBodyInstancesInstancePublicIpAddress) string {
	fip := ""
	if floatingIp != nil {
		fip = utils.JoinStringPtr(floatingIp.IpAddress, "")
	}
	if fip == "" && eIp != nil {
		return utils.SafeString(eIp.IpAddress)
	}
	return fip
}
