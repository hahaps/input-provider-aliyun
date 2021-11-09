package network

import (
	"errors"
	"fmt"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/hahaps/common-provider/src/common/utils"
	"github.com/hahaps/common-provider/src/input"
	"github.com/hahaps/common-provider/src/models/network"
	"github.com/hahaps/input-provider-aliyun/src/common"
	"strconv"
)

var NicSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "region",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "subnet",
		Required: false,
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

type Nic struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (nic *Nic)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	nic.credential = credential
	nic.client = cli
	return nil
}

func (Nic)Call(params input.Params, replay *input.Replay) error {
	nic := &Nic{}
	var next string
	var err error
	var nics []interface{}
	params.Args, err = utils.CheckParam(params.Args, NicSchemes)
	if err != nil {
		return err
	}
	err = nic.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	sub := ""
	if params.Args["subnet"] != nil {
		sub = params.Args["subnet"].(string)
	}
	region := params.Args["region"].(string)
	if next == "" {
		return nil
	}
	number, err := strconv.ParseInt(next,10,32)
	if err != nil {
		return errors.New("bad page number[marker] info")
	}
	pageNum := int32(number)
	query := map[string]interface{} {
		"RegionId": region,
		"CloudType": common.CloudType,
		"AccountId": nic.credential.AccountId,
	}
	request := &ecs20140526.DescribeNetworkInterfacesRequest{
		RegionId: &region,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	if sub != "" {
		request.VSwitchId = &sub
	}
	resp, err := nic.client.DescribeNetworkInterfaces(request)
	if err != nil {
		return err
	}
	if resp.Body.NetworkInterfaceSets == nil || resp.Body.NetworkInterfaceSets.NetworkInterfaceSet == nil {
		return errors.New("bad response for query vSwitch")
	}
	for _, b := range resp.Body.NetworkInterfaceSets.NetworkInterfaceSet {
		s := network.NewNicModel()
		s.Deleted = 0
		s.CloudType = common.CloudType
		s.AccountId = nic.credential.AccountId
		s.RegionId = region
		s.Name = utils.SafeString(b.NetworkInterfaceName)
		s.ProviderId = utils.SafeString(b.NetworkInterfaceId)
		s.CIDR = utils.SafeString(b.PrivateIpAddress)
		s.Status = utils.SafeString(b.Status)
		s.CreateTime = utils.SafeString(b.CreationTime)
		s.Description = utils.SafeString(b.Description)
		s.NetworkId = utils.SafeString(b.VpcId)
		s.SubnetId = utils.SafeString(b.VSwitchId)
		s.Tags = getNicTags(b.Tags)
		s.InstanceId = utils.SafeString(b.InstanceId)
		s.Extra = map[string]interface{}{
			"ZoneId": utils.SafeString(b.ZoneId),
			"ResourceGroupId": utils.SafeString(b.ResourceGroupId),
			"Type": utils.SafeString(b.Type),
			"MacAddress": utils.SafeString(b.MacAddress),
		}
		if b.AssociatedPublicIp != nil {
			s.Extra["FloatingIp"] = b.AssociatedPublicIp.PublicIpAddress
		}
		s.SetIndex()
		s.SetChecksum()
		checked, key := s.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		nics = append(nics, s)
	}
	if !utils.CheckQueryKeys(query, network.NicModel{}) {
		return errors.New("query key is not attribute of NicModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = nics
	return nil
}

func getNicTags(tags *ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSetTags) string {
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
