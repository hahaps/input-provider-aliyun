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

var NetworkSchemes = []utils.Scheme {
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

type Network struct {
	client *ecs20140526.Client
	credential input.Credential
	input.Resource
}

func (vpc *Network)init(credential input.Credential) error {
	config := &openapi.Config{
		AccessKeyId: &credential.SecretId,
		AccessKeySecret: &credential.SecretKey,
	}
	config.Endpoint = tea.String("ecs.aliyuncs.com")
	cli, err := ecs20140526.NewClient(config)
	if err != nil {
		return err
	}
	vpc.credential = credential
	vpc.client = cli
	return nil
}

func (Network)Call(params input.Params, replay *input.Replay) error {
	vpc := &Network{}
	var next string
	var err error
	var nets []interface{}
	params.Args, err = utils.CheckParam(params.Args, NetworkSchemes)
	if err != nil {
		return err
	}
	err = vpc.init(params.Credential)
	if err != nil {
		return err
	}
	limit := int32(params.Args["limit"].(int))
	next = params.Args["marker"].(string)
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
		"AccountId": vpc.credential.AccountId,
	}
	request := &ecs20140526.DescribeVpcsRequest{
		RegionId: &region,
		PageSize: &limit,
		PageNumber: &pageNum,
	}
	resp, err := vpc.client.DescribeVpcs(request)
	if err != nil {
		return err
	}
	if resp.Body.Vpcs == nil || resp.Body.Vpcs.Vpc == nil {
		return errors.New("bad response for query networks")
	}
	for _, b := range resp.Body.Vpcs.Vpc {
		net := network.NewNetworkModel()
		net.Deleted = 0
		net.CloudType = common.CloudType
		net.AccountId = vpc.credential.AccountId
		net.RegionId = region
		net.Name = utils.SafeString(b.VpcName)
		net.ProviderId = utils.SafeString(b.VpcId)
		net.Type = "vpc"
		net.Status = utils.SafeString(b.Status)
		net.CreateTime = utils.SafeString(b.CreationTime)
		net.Description = utils.SafeString(b.Description)
		net.CIDR = utils.SafeString(b.CidrBlock)
		net.Extra = map[string]interface{}{
			"VRouterId": utils.SafeString(b.VRouterId),
		}
		net.SetIndex()
		net.SetChecksum()
		checked, key := net.CheckRequired()
		if !checked {
			return errors.New(
				fmt.Sprintf("Value[%v] should not be empty", key))
		}
		nets = append(nets, net)
	}
	if !utils.CheckQueryKeys(query, network.NetworkModel{}) {
		return errors.New("query key is not attribute of NetworkModel")
	}
	total := utils.SafeInt32(resp.Body.TotalCount)
	if pageNum * limit >= total {
		next = ""
	} else {
		next = strconv.Itoa(int(pageNum + 1))
	}
	replay.Next = next
	replay.Query = query
	replay.Result = nets
	return nil
}

