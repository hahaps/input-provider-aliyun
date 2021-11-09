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

var SubnetSchemes = []utils.Scheme {
	utils.Scheme{
		Param: "region",
		Required: true,
		Type: utils.String,
	},
	utils.Scheme{
		Param: "network",
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

type Subnet struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (sub *Subnet)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	sub.credential = credential
	sub.client = cli
	return nil
}

func (Subnet)Call(params input.Params, replay *input.Replay) error {
	sub := &Subnet{}
	var next string
	var err error
	var subnets []interface{}
	params.Args, err = utils.CheckParam(params.Args, NetworkSchemes)
	if err != nil {
		return err
	}
	err = sub.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
	net := ""
	if params.Args["network"] != nil {
		net = params.Args["network"].(string)
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
		"AccountId": sub.credential.AccountId,
	}
	request := &ecs20140526.DescribeVSwitchesRequest{
		RegionId: &region,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	if net != "" {
		request.VpcId = &net
	}
	resp, err := sub.client.DescribeVSwitches(request)
	if err != nil {
		return err
	}
	if resp.Body.VSwitches == nil || resp.Body.VSwitches.VSwitch == nil {
		return errors.New("bad response for query vSwitch")
	}
	for _, b := range resp.Body.VSwitches.VSwitch {
		s := network.NewSubnetModel()
		s.Deleted = 0
		s.CloudType = common.CloudType
		s.AccountId = sub.credential.AccountId
		s.RegionId = region
		s.Name = utils.SafeString(b.VSwitchName)
		s.ProviderId = utils.SafeString(b.VSwitchId)
		s.CIDR = utils.SafeString(b.CidrBlock)
		s.Status = utils.SafeString(b.Status)
		s.CreateTime = utils.SafeString(b.CreationTime)
		s.Description = utils.SafeString(b.Description)
		s.NetworkId = utils.SafeString(b.VpcId)
		s.Extra = map[string]interface{}{
			"ZoneId": utils.SafeString(b.ZoneId),
			"ResourceGroupId": utils.SafeString(b.ResourceGroupId),
		}
		s.SetIndex()
		s.SetChecksum()
		checked, key := s.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		subnets = append(subnets, s)
	}
	if !utils.CheckQueryKeys(query, network.NetworkModel{}) {
		return errors.New("query key is not attribute of SubnetModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = subnets
	return nil
}


